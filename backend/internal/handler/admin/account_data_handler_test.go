package admin

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dataResponse struct {
	Code int         `json:"code"`
	Data dataPayload `json:"data"`
}

type dataPayload struct {
	Type     string        `json:"type"`
	Version  int           `json:"version"`
	Proxies  []dataProxy   `json:"proxies"`
	Accounts []dataAccount `json:"accounts"`
}

type dataProxy struct {
	ProxyKey string `json:"proxy_key"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type dataAccount struct {
	Name        string         `json:"name"`
	Platform    string         `json:"platform"`
	Type        string         `json:"type"`
	Credentials map[string]any `json:"credentials"`
	Extra       map[string]any `json:"extra"`
	ProxyKey    *string        `json:"proxy_key"`
	Concurrency int            `json:"concurrency"`
	Priority    int            `json:"priority"`
}

func setupAccountDataRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()

	h := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router.GET("/api/v1/admin/accounts/data", h.ExportData)
	router.POST("/api/v1/admin/accounts/data", h.ImportData)
	router.POST("/api/v1/admin/accounts/data/upload", h.ImportDataUpload)
	return router, adminSvc
}

type dataImportResultResponse struct {
	Code int              `json:"code"`
	Data DataImportResult `json:"data"`
}

func TestExportDataIncludesSecrets(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	proxyID := int64(11)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       proxyID,
			Name:     "proxy",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
		{
			ID:       12,
			Name:     "orphan",
			Protocol: "https",
			Host:     "10.0.0.1",
			Port:     443,
			Username: "o",
			Password: "p",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			Extra:       map[string]any{"note": "x"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			Priority:    50,
			Status:      service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Empty(t, resp.Data.Type)
	require.Equal(t, 0, resp.Data.Version)
	require.Len(t, resp.Data.Proxies, 1)
	require.Equal(t, "pass", resp.Data.Proxies[0].Password)
	require.Len(t, resp.Data.Accounts, 1)
	require.Equal(t, "secret", resp.Data.Accounts[0].Credentials["token"])
}

func TestExportDataWithoutProxies(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	proxyID := int64(11)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       proxyID,
			Name:     "proxy",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			Priority:    50,
			Status:      service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data?include_proxies=false", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Proxies, 0)
	require.Len(t, resp.Data.Accounts, 1)
	require.Nil(t, resp.Data.Accounts[0].ProxyKey)
}

func TestExportDataPassesAccountFiltersAndSort(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "acc-1", Status: service.StatusActive},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/accounts/data?platform=openai&type=oauth&status=active&group=12&privacy_mode=blocked&search=keyword&sort_by=priority&sort_order=desc",
		nil,
	)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, 1, adminSvc.lastListAccounts.calls)
	require.Equal(t, "openai", adminSvc.lastListAccounts.platform)
	require.Equal(t, "oauth", adminSvc.lastListAccounts.accountType)
	require.Equal(t, "active", adminSvc.lastListAccounts.status)
	require.Equal(t, int64(12), adminSvc.lastListAccounts.groupID)
	require.Equal(t, "blocked", adminSvc.lastListAccounts.privacyMode)
	require.Equal(t, "keyword", adminSvc.lastListAccounts.search)
	require.Equal(t, "priority", adminSvc.lastListAccounts.sortBy)
	require.Equal(t, "desc", adminSvc.lastListAccounts.sortOrder)
}

func TestExportDataSelectedIDsOverrideFilters(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/admin/accounts/data?ids=1,2&platform=openai&search=keyword&sort_by=priority&sort_order=desc",
		nil,
	)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Accounts, 2)
	require.Equal(t, 0, adminSvc.lastListAccounts.calls)
}

func TestImportDataReusesProxyAndSkipsDefaultGroup(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy",
			Protocol: "socks5",
			Host:     "1.2.3.4",
			Port:     1080,
			Username: "u",
			Password: "p",
			Status:   service.StatusActive,
		},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key": "socks5|1.2.3.4|1080|u|p",
					"name":      "proxy",
					"protocol":  "socks5",
					"host":      "1.2.3.4",
					"port":      1080,
					"username":  "u",
					"password":  "p",
					"status":    "active",
				},
			},
			"accounts": []map[string]any{
				{
					"name":        "acc",
					"platform":    service.PlatformOpenAI,
					"type":        service.AccountTypeOAuth,
					"credentials": map[string]any{"token": "x"},
					"proxy_key":   "socks5|1.2.3.4|1080|u|p",
					"concurrency": 3,
					"priority":    50,
				},
			},
		},
		"skip_default_group_bind": true,
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdProxies, 0)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.True(t, adminSvc.createdAccounts[0].SkipDefaultGroupBind)
}

func TestImportDataUploadConvertsCPAZipAndAppliesExplicitTemplateAccount(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	rateMultiplier := 1.75
	adminSvc.accounts = []service.Account{
		{
			ID:                 999,
			Name:               "template@outlook.com",
			Platform:           service.PlatformOpenAI,
			Type:               service.AccountTypeOAuth,
			GroupIDs:           []int64{2, 3, 4, 5},
			Concurrency:        7,
			Priority:           9,
			RateMultiplier:     &rateMultiplier,
			AutoPauseOnExpired: false,
			Status:             service.StatusActive,
		},
	}

	zipBytes := buildTestCPAZip(t, map[string]cpaOAuthSource{
		"oauth/sub_demo_outlook.com.json": {
			Email:        "demo@outlook.com",
			AccountID:    "acc-123",
			AccessToken:  buildTestJWT(map[string]any{"exp": 1778069305, "client_id": openai.ClientID}),
			RefreshToken: "rt_demo",
			IDToken: buildTestJWT(map[string]any{
				"email": "demo@outlook.com",
				"name":  "Demo User",
				"https://api.openai.com/auth": map[string]any{
					"chatgpt_account_id":                "acc-123",
					"chatgpt_user_id":                   "user-123",
					"chatgpt_plan_type":                 "plus",
					"chatgpt_subscription_active_until": "2026-05-26T11:43:19+00:00",
					"organizations": []map[string]any{
						{"id": "org-123", "is_default": true},
					},
				},
			}),
			GeneratedAt: "2026-04-26T12:08:25Z",
			OAuthTokenResponse: map[string]any{
				"token_type": "bearer",
				"expires_in": 864000,
				"scope":      "openid profile email offline_access",
			},
		},
	})

	body, contentType := buildMultipartUpload(t, "file", "masters.zip", zipBytes, map[string]string{
		"include_email_domains":   "outlook.com",
		"skip_default_group_bind": "true",
		"template_account_id":     "999",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data/upload", body)
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	created := adminSvc.createdAccounts[0]
	require.Equal(t, "demo@outlook.com", created.Name)
	require.Equal(t, service.PlatformOpenAI, created.Platform)
	require.Equal(t, service.AccountTypeOAuth, created.Type)
	require.True(t, created.SkipDefaultGroupBind)
	require.Equal(t, []int64{2, 3, 4, 5}, created.GroupIDs)
	require.Equal(t, 7, created.Concurrency)
	require.Equal(t, 9, created.Priority)
	require.NotNil(t, created.RateMultiplier)
	require.Equal(t, 1.75, *created.RateMultiplier)
	require.NotNil(t, created.AutoPauseOnExpired)
	require.False(t, *created.AutoPauseOnExpired)
	require.Equal(t, "training_off", created.Extra["privacy_mode"])
	require.Equal(t, true, created.Extra["openai_passthrough"])
	require.Equal(t, "plus", created.Credentials["plan_type"])
	require.Equal(t, "Imported from masters.zip", *created.Notes)

	var resp dataImportResultResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "cpa-zip", resp.Data.SourceFormat)
	require.Equal(t, 1, resp.Data.SourceAccountTotal)
	require.Equal(t, 1, resp.Data.AccountCreated)
	require.Equal(t, 0, resp.Data.AccountSkippedExisting)
}

func TestImportDataUploadFiltersDomainsAndSkipsExisting(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = []service.Account{
		{
			ID:       998,
			Name:     "exist@outlook.com",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
		},
	}

	zipBytes := buildTestCPAZip(t, map[string]cpaOAuthSource{
		"oauth/exist_outlook.json": {
			Email:        "exist@outlook.com",
			AccountID:    "acc-exist",
			AccessToken:  buildTestJWT(map[string]any{"exp": 1778069305, "client_id": openai.ClientID}),
			RefreshToken: "rt_exist",
			IDToken:      buildTestJWT(map[string]any{"email": "exist@outlook.com"}),
			GeneratedAt:  "2026-04-26T12:08:25Z",
		},
		"oauth/other_domain.json": {
			Email:        "other@example.com",
			AccountID:    "acc-other",
			AccessToken:  buildTestJWT(map[string]any{"exp": 1778069305, "client_id": openai.ClientID}),
			RefreshToken: "rt_other",
			IDToken:      buildTestJWT(map[string]any{"email": "other@example.com"}),
			GeneratedAt:  "2026-04-26T12:08:25Z",
		},
		"oauth/new_outlook.json": {
			Email:        "new@outlook.com",
			AccountID:    "acc-new",
			AccessToken:  buildTestJWT(map[string]any{"exp": 1778069305, "client_id": openai.ClientID}),
			RefreshToken: "rt_new",
			IDToken:      buildTestJWT(map[string]any{"email": "new@outlook.com"}),
			GeneratedAt:  "2026-04-26T12:08:25Z",
		},
	})

	body, contentType := buildMultipartUpload(t, "file", "masters.zip", zipBytes, map[string]string{
		"include_email_domains": "outlook.com",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data/upload", body)
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdAccounts, 1)
	require.Equal(t, "new@outlook.com", adminSvc.createdAccounts[0].Name)

	var resp dataImportResultResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 1, resp.Data.AccountCreated)
	require.Equal(t, 1, resp.Data.AccountSkippedExisting)
	require.Equal(t, 2, resp.Data.SourceAccountFiltered)
}

func buildMultipartUpload(t *testing.T, fieldName, fileName string, content []byte, formFields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range formFields {
		require.NoError(t, writer.WriteField(key, value))
	}
	part, err := writer.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return &body, writer.FormDataContentType()
}

func buildTestCPAZip(t *testing.T, files map[string]cpaOAuthSource) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, payload := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		_, err = w.Write(raw)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func buildTestJWT(payload map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	body, _ := json.Marshal(payload)
	claims := base64.RawURLEncoding.EncodeToString(body)
	return strings.Join([]string{header, claims, "sig"}, ".")
}
