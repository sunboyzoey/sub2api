package admin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const dataImportUploadMaxBytes = 8 << 20

type dataUploadParseMeta struct {
	sourceFormat       string
	sourceAccountTotal int
	sourceFiltered     int
	sourceInvalid      int
	preImportErrors    []DataImportError
}

type cpaOAuthSource struct {
	Email              string         `json:"email"`
	Expired            string         `json:"expired"`
	IDToken            string         `json:"id_token"`
	AccountID          string         `json:"account_id"`
	AccessToken        string         `json:"access_token"`
	RefreshToken       string         `json:"refresh_token"`
	GeneratedAt        string         `json:"generated_at"`
	OAuthTokenResponse map[string]any `json:"oauth_token_response"`
}

type cpaSourceEntry struct {
	fileName    string
	email       string
	generatedAt int64
	source      cpaOAuthSource
}

type uploadTemplateDefaults struct {
	AccountID          int64
	Name               string
	GroupIDs           []int64
	Concurrency        int
	Priority           int
	RateMultiplier     *float64
	AutoPauseOnExpired *bool
}

// ImportDataUpload accepts either sub2api export JSON or CPA OAuth JSON/ZIP and imports it.
func (h *AccountHandler) ImportDataUpload(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(dataImportUploadMaxBytes); err != nil {
		response.BadRequest(c, "Invalid upload: "+err.Error())
		return
	}

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(io.LimitReader(file, dataImportUploadMaxBytes+1))
	if err != nil {
		response.BadRequest(c, "Failed to read uploaded file")
		return
	}
	if len(raw) > dataImportUploadMaxBytes {
		response.BadRequest(c, fmt.Sprintf("uploaded file exceeds %d bytes", dataImportUploadMaxBytes))
		return
	}

	skipDefaultGroupBind, err := parseUploadBoolWithDefault(c.PostForm("skip_default_group_bind"), true)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	includeDomains := parseDomainList(c.PostForm("include_email_domains"))
	templateAccountID, err := parseOptionalInt64(c.PostForm("template_account_id"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	payload, meta, err := parseUploadedDataFile(fileHeader.Filename, raw, includeDomains)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := validateDataHeader(payload); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if meta.sourceFormat == "cpa-json" || meta.sourceFormat == "cpa-zip" {
		templateDefaults, groupErr := h.resolveUploadTemplateDefaults(c.Request.Context(), templateAccountID, service.PlatformOpenAI, service.AccountTypeOAuth)
		if groupErr != nil {
			response.BadRequest(c, groupErr.Error())
			return
		}
		if templateDefaults != nil {
			applyUploadTemplateDefaults(payload.Accounts, templateDefaults)
		}

		existingEmails, listErr := h.collectExistingOpenAIOAuthEmails(c.Request.Context())
		if listErr != nil {
			response.BadRequest(c, "Failed to inspect existing OpenAI OAuth accounts")
			return
		}
		filteredAccounts := make([]DataAccount, 0, len(payload.Accounts))
		skippedExisting := 0
		for _, account := range payload.Accounts {
			email := extractDataAccountEmail(account)
			if email != "" {
				if _, ok := existingEmails[email]; ok {
					skippedExisting++
					continue
				}
			}
			filteredAccounts = append(filteredAccounts, account)
		}
		payload.Accounts = filteredAccounts
		meta.sourceFiltered += skippedExisting

		req := DataImportRequest{
			Data:                 payload,
			SkipDefaultGroupBind: &skipDefaultGroupBind,
		}
		idempotencyPayload := struct {
			FileName             string      `json:"file_name"`
			SkipDefaultGroupBind bool        `json:"skip_default_group_bind"`
			IncludeEmailDomains  []string    `json:"include_email_domains,omitempty"`
			TemplateAccountID    *int64      `json:"template_account_id,omitempty"`
			Data                 DataPayload `json:"data"`
			SourceFormat         string      `json:"source_format"`
		}{
			FileName:             fileHeader.Filename,
			SkipDefaultGroupBind: skipDefaultGroupBind,
			IncludeEmailDomains:  includeDomains,
			TemplateAccountID:    templateAccountID,
			Data:                 payload,
			SourceFormat:         meta.sourceFormat,
		}

		executeAdminIdempotentJSON(c, "admin.accounts.import_data_upload", idempotencyPayload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
			result, importErr := h.importData(ctx, req)
			if importErr != nil {
				return nil, importErr
			}
			result.AccountSkippedExisting = skippedExisting
			result.SourceFormat = meta.sourceFormat
			result.SourceAccountTotal = meta.sourceAccountTotal
			result.SourceAccountFiltered = meta.sourceFiltered
			result.SourceAccountInvalid = meta.sourceInvalid
			if len(meta.preImportErrors) > 0 {
				result.AccountFailed += len(meta.preImportErrors)
				result.Errors = append(slices.Clone(meta.preImportErrors), result.Errors...)
			}
			return result, nil
		})
		return
	}

	req := DataImportRequest{
		Data:                 payload,
		SkipDefaultGroupBind: &skipDefaultGroupBind,
	}
	executeAdminIdempotentJSON(c, "admin.accounts.import_data_upload", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		result, importErr := h.importData(ctx, req)
		if importErr != nil {
			return nil, importErr
		}
		result.SourceFormat = meta.sourceFormat
		return result, nil
	})
}

func parseUploadBoolWithDefault(raw string, defaultValue bool) (bool, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return defaultValue, fmt.Errorf("invalid boolean value: %s", raw)
	}
}

func parseOptionalInt64(raw string) (*int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return nil, fmt.Errorf("invalid template_account_id: %s", raw)
	}
	return &id, nil
}

func parseDomainList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ';'
	}) {
		domain := strings.TrimSpace(strings.ToLower(part))
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}
	return out
}

func parseUploadedDataFile(fileName string, raw []byte, includeDomains []string) (DataPayload, dataUploadParseMeta, error) {
	if payload, ok := tryParseDataPayloadJSON(raw); ok {
		return payload, dataUploadParseMeta{sourceFormat: "sub2api-json"}, nil
	}

	baseName := filepath.Base(fileName)
	if account, ok := tryParseCPAOAuthJSON(raw); ok {
		payload, meta, err := convertCPAEntriesToPayload(baseName, []cpaSourceEntry{
			{
				fileName:    baseName,
				email:       normalizeEmail(account.Email),
				generatedAt: parseRFC3339Unix(account.GeneratedAt),
				source:      account,
			},
		}, includeDomains)
		if err != nil {
			return DataPayload{}, dataUploadParseMeta{}, err
		}
		meta.sourceFormat = "cpa-json"
		return payload, meta, nil
	}

	if isZipArchive(raw) {
		reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			return DataPayload{}, dataUploadParseMeta{}, fmt.Errorf("failed to parse zip: %w", err)
		}
		payload, meta, err := convertCPAZipToPayload(baseName, reader, includeDomains)
		if err != nil {
			return DataPayload{}, dataUploadParseMeta{}, err
		}
		meta.sourceFormat = "cpa-zip"
		return payload, meta, nil
	}

	return DataPayload{}, dataUploadParseMeta{}, fmt.Errorf("unsupported file format: %s", baseName)
}

func tryParseDataPayloadJSON(raw []byte) (DataPayload, bool) {
	var payload DataPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return DataPayload{}, false
	}
	if payload.Accounts == nil || payload.Proxies == nil {
		return DataPayload{}, false
	}
	if err := validateDataHeader(payload); err != nil {
		return DataPayload{}, false
	}
	return payload, true
}

func tryParseCPAOAuthJSON(raw []byte) (cpaOAuthSource, bool) {
	var source cpaOAuthSource
	if err := json.Unmarshal(raw, &source); err != nil {
		return cpaOAuthSource{}, false
	}
	if normalizeEmail(source.Email) == "" || strings.TrimSpace(source.AccessToken) == "" || strings.TrimSpace(source.RefreshToken) == "" {
		return cpaOAuthSource{}, false
	}
	return source, true
}

func isZipArchive(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == 'P' && raw[1] == 'K' && raw[2] == 0x03 && raw[3] == 0x04
}

func convertCPAZipToPayload(fileName string, reader *zip.Reader, includeDomains []string) (DataPayload, dataUploadParseMeta, error) {
	entries := make([]cpaSourceEntry, 0)
	meta := dataUploadParseMeta{}
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "oauth/") || !strings.HasSuffix(strings.ToLower(file.Name), ".json") {
			continue
		}
		meta.sourceAccountTotal++
		rc, err := file.Open()
		if err != nil {
			meta.sourceInvalid++
			meta.preImportErrors = append(meta.preImportErrors, DataImportError{
				Kind:    "account",
				Name:    filepath.Base(file.Name),
				Message: "failed to open zip entry",
			})
			continue
		}
		data, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			meta.sourceInvalid++
			meta.preImportErrors = append(meta.preImportErrors, DataImportError{
				Kind:    "account",
				Name:    filepath.Base(file.Name),
				Message: "failed to read zip entry",
			})
			continue
		}
		source, ok := tryParseCPAOAuthJSON(data)
		if !ok {
			meta.sourceInvalid++
			meta.preImportErrors = append(meta.preImportErrors, DataImportError{
				Kind:    "account",
				Name:    filepath.Base(file.Name),
				Message: "invalid CPA OAuth json",
			})
			continue
		}
		entries = append(entries, cpaSourceEntry{
			fileName:    filepath.Base(file.Name),
			email:       normalizeEmail(source.Email),
			generatedAt: parseRFC3339Unix(source.GeneratedAt),
			source:      source,
		})
	}
	if meta.sourceAccountTotal == 0 {
		return DataPayload{}, meta, fmt.Errorf("zip does not contain oauth/*.json entries")
	}
	payload, convertedMeta, err := convertCPAEntriesToPayload(fileName, entries, includeDomains)
	if err != nil {
		return DataPayload{}, meta, err
	}
	meta.sourceFiltered += convertedMeta.sourceFiltered
	meta.sourceInvalid += convertedMeta.sourceInvalid
	meta.preImportErrors = append(meta.preImportErrors, convertedMeta.preImportErrors...)
	return payload, meta, nil
}

func convertCPAEntriesToPayload(fileName string, entries []cpaSourceEntry, includeDomains []string) (DataPayload, dataUploadParseMeta, error) {
	includeSet := make(map[string]struct{}, len(includeDomains))
	for _, domain := range includeDomains {
		includeSet[strings.ToLower(strings.TrimSpace(domain))] = struct{}{}
	}

	meta := dataUploadParseMeta{sourceAccountTotal: len(entries)}
	deduped := make(map[string]cpaSourceEntry, len(entries))

	for _, entry := range entries {
		if entry.email == "" {
			meta.sourceInvalid++
			meta.preImportErrors = append(meta.preImportErrors, DataImportError{
				Kind:    "account",
				Name:    entry.fileName,
				Message: "missing email",
			})
			continue
		}
		if len(includeSet) > 0 {
			if _, ok := includeSet[emailDomain(entry.email)]; !ok {
				meta.sourceFiltered++
				continue
			}
		}

		existing, ok := deduped[entry.email]
		if !ok || shouldReplaceCPAEntry(existing, entry) {
			deduped[entry.email] = entry
		}
	}

	if len(deduped) == 0 {
		return DataPayload{}, meta, fmt.Errorf("no CPA accounts matched the selected filters")
	}

	noteText := fmt.Sprintf("Imported from %s", fileName)
	rateMultiplier := 1.0
	autoPause := true
	accounts := make([]DataAccount, 0, len(deduped))
	emails := make([]string, 0, len(deduped))
	for email := range deduped {
		emails = append(emails, email)
	}
	slices.Sort(emails)

	for _, email := range emails {
		entry := deduped[email]
		credentials, extra := buildCPACredentials(entry.source)
		accounts = append(accounts, DataAccount{
			Name:               email,
			Notes:              stringPtr(noteText),
			Platform:           service.PlatformOpenAI,
			Type:               service.AccountTypeOAuth,
			Credentials:        credentials,
			Extra:              extra,
			Concurrency:        10,
			Priority:           1,
			RateMultiplier:     &rateMultiplier,
			AutoPauseOnExpired: &autoPause,
		})
	}

	return DataPayload{
		Type:       dataType,
		Version:    dataVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Proxies:    []DataProxy{},
		Accounts:   accounts,
	}, meta, nil
}

func shouldReplaceCPAEntry(current, candidate cpaSourceEntry) bool {
	if candidate.generatedAt > current.generatedAt {
		return true
	}
	if candidate.generatedAt == current.generatedAt && candidate.fileName < current.fileName {
		return true
	}
	return false
}

func buildCPACredentials(source cpaOAuthSource) (map[string]any, map[string]any) {
	email := normalizeEmail(source.Email)
	idClaims := decodeJWTMap(source.IDToken)
	accessClaims := decodeJWTMap(source.AccessToken)
	authClaims := jwtMapField(idClaims, "https://api.openai.com/auth")
	if len(authClaims) == 0 {
		authClaims = jwtMapField(accessClaims, "https://api.openai.com/auth")
	}
	profileClaims := jwtMapField(accessClaims, "https://api.openai.com/profile")

	if email == "" {
		email = normalizeEmail(stringField(idClaims, "email"))
	}
	if email == "" {
		email = normalizeEmail(stringField(profileClaims, "email"))
	}

	clientID := stringField(accessClaims, "client_id")
	if clientID == "" {
		clientID = firstString(stringSliceField(idClaims, "aud"))
	}
	if clientID == "" {
		clientID = openai.ClientID
	}

	expiresAt := intField(accessClaims, "exp")
	if expiresAt == 0 {
		expiresAt = parseRFC3339Unix(source.Expired)
	}
	organizationID := defaultOrganizationID(authClaims)

	credentials := map[string]any{
		"access_token":  strings.TrimSpace(source.AccessToken),
		"refresh_token": strings.TrimSpace(source.RefreshToken),
		"client_id":     clientID,
		"expires_at":    expiresAt,
	}
	setIfNotEmpty(credentials, "id_token", strings.TrimSpace(source.IDToken))
	setIfNotEmpty(credentials, "email", email)
	setIfNotEmpty(credentials, "chatgpt_account_id", firstNonEmptyUpload(strings.TrimSpace(source.AccountID), stringField(authClaims, "chatgpt_account_id")))
	setIfNotEmpty(credentials, "chatgpt_user_id", stringField(authClaims, "chatgpt_user_id"))
	setIfNotEmpty(credentials, "organization_id", organizationID)
	setIfNotEmpty(credentials, "plan_type", stringField(authClaims, "chatgpt_plan_type"))
	setIfNotEmpty(credentials, "subscription_expires_at", stringField(authClaims, "chatgpt_subscription_active_until"))
	setIfNotEmpty(credentials, "token_type", stringField(source.OAuthTokenResponse, "token_type"))
	if expiresIn := intField(source.OAuthTokenResponse, "expires_in"); expiresIn > 0 {
		credentials["expires_in"] = expiresIn
	}
	setIfNotEmpty(credentials, "scope", stringField(source.OAuthTokenResponse, "scope"))

	extra := map[string]any{
		"email":              email,
		"privacy_mode":       "training_off",
		"openai_passthrough": true,
	}
	setIfNotEmpty(extra, "name", stringField(idClaims, "name"))

	return credentials, extra
}

func decodeJWTMap(token string) map[string]any {
	token = strings.TrimSpace(token)
	if token == "" {
		return map[string]any{}
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return map[string]any{}
	}
	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return map[string]any{}
		}
	}
	var out map[string]any
	if err := json.Unmarshal(decoded, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func jwtMapField(claims map[string]any, key string) map[string]any {
	value, ok := claims[key]
	if !ok {
		return map[string]any{}
	}
	out, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return out
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

func stringSliceField(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	value, ok := m[key]
	if !ok || value == nil {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func intField(m map[string]any, key string) int64 {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		i, _ := v.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i
	default:
		return 0
	}
}

func defaultOrganizationID(authClaims map[string]any) string {
	orgs, ok := authClaims["organizations"].([]any)
	if ok {
		for _, item := range orgs {
			org, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if isDefault, _ := org["is_default"].(bool); isDefault {
				if id := stringField(org, "id"); id != "" {
					return id
				}
			}
		}
		for _, item := range orgs {
			org, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if id := stringField(org, "id"); id != "" {
				return id
			}
		}
	}
	if poid := stringField(authClaims, "poid"); poid != "" {
		return poid
	}
	return ""
}

func firstNonEmptyUpload(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func setIfNotEmpty(target map[string]any, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	target[key] = strings.TrimSpace(value)
}

func parseRFC3339Unix(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	ts, err := time.Parse(time.RFC3339, strings.ReplaceAll(value, "Z", "+00:00"))
	if err != nil {
		return 0
	}
	return ts.Unix()
}

func stringPtr(value string) *string {
	return &value
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func emailDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parts[1]))
}

func extractDataAccountEmail(account DataAccount) string {
	if email := normalizeEmail(account.Name); strings.Contains(email, "@") {
		return email
	}
	if email := normalizeEmail(stringField(account.Credentials, "email")); email != "" {
		return email
	}
	if email := normalizeEmail(stringField(account.Extra, "email")); email != "" {
		return email
	}
	return ""
}

func (h *AccountHandler) collectExistingOpenAIOAuthEmails(ctx context.Context) (map[string]struct{}, error) {
	accounts, err := h.listAccountsFiltered(ctx, service.PlatformOpenAI, service.AccountTypeOAuth, "", "", 0, "", "created_at", "desc")
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		for _, email := range []string{
			normalizeEmail(account.Name),
			normalizeEmail(account.GetCredential("email")),
			normalizeEmail(stringField(account.Extra, "email")),
		} {
			if email == "" || !strings.Contains(email, "@") {
				continue
			}
			out[email] = struct{}{}
		}
	}
	return out, nil
}

func (h *AccountHandler) resolveUploadTemplateDefaults(ctx context.Context, templateAccountID *int64, platform, accountType string) (*uploadTemplateDefaults, error) {
	if templateAccountID != nil {
		account, err := h.adminService.GetAccount(ctx, *templateAccountID)
		if err != nil || account == nil {
			return nil, fmt.Errorf("template account not found")
		}
		return buildUploadTemplateDefaults(account)
	}

	accounts, err := h.listAccountsFiltered(ctx, platform, accountType, "", "", 0, "", "created_at", "desc")
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if len(account.GroupIDs) == 0 {
			continue
		}
		cloned := account
		return buildUploadTemplateDefaults(&cloned)
	}
	return nil, nil
}

func buildUploadTemplateDefaults(account *service.Account) (*uploadTemplateDefaults, error) {
	if account == nil {
		return nil, nil
	}
	if len(account.GroupIDs) == 0 {
		return nil, fmt.Errorf("template account has no groups")
	}
	return &uploadTemplateDefaults{
		AccountID:          account.ID,
		Name:               account.Name,
		GroupIDs:           slices.Clone(account.GroupIDs),
		Concurrency:        account.Concurrency,
		Priority:           account.Priority,
		RateMultiplier:     cloneFloat64Ptr(account.RateMultiplier),
		AutoPauseOnExpired: boolPtr(account.AutoPauseOnExpired),
	}, nil
}

func applyUploadTemplateDefaults(accounts []DataAccount, defaults *uploadTemplateDefaults) {
	if defaults == nil {
		return
	}
	for i := range accounts {
		accounts[i].GroupIDs = slices.Clone(defaults.GroupIDs)
		if defaults.Concurrency > 0 {
			accounts[i].Concurrency = defaults.Concurrency
		}
		accounts[i].Priority = defaults.Priority
		accounts[i].RateMultiplier = cloneFloat64Ptr(defaults.RateMultiplier)
		accounts[i].AutoPauseOnExpired = boolPtr(derefBool(defaults.AutoPauseOnExpired))
	}
}

func cloneFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func boolPtr(value bool) *bool {
	return &value
}

func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}
