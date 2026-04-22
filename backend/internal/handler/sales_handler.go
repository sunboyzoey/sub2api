package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	salesPartnerSecretHeader = "X-Sales-Partner-Secret"
	salesAuthorizationHeader = "Authorization"
)

type SalesHandler struct {
	salesService *service.SalesService
}

func NewSalesHandler(salesService *service.SalesService) *SalesHandler {
	return &SalesHandler{salesService: salesService}
}

type salesPartnerProvisionRequest struct {
	ExternalOrderID     string         `json:"external_order_id" binding:"required,max=100"`
	ExternalUserID      string         `json:"external_user_id" binding:"omitempty,max=100"`
	ExternalEmail       string         `json:"external_email" binding:"omitempty,email"`
	ExternalName        string         `json:"external_name"`
	APIKey              string         `json:"api_key" binding:"omitempty,max=128"`
	PackageID           int64          `json:"package_id"`
	ExternalPackageCode string         `json:"external_package_code"`
	OrderType           string         `json:"order_type" binding:"omitempty,oneof=purchase renewal manual"`
	Amount              float64        `json:"amount" binding:"omitempty,min=0"`
	Currency            string         `json:"currency" binding:"omitempty,max=16"`
	RawPayload          map[string]any `json:"raw_payload"`
}

func (h *SalesHandler) ListStoreCatalog(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, pag, err := h.salesService.ListStoreCatalog(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesCatalogFilters{
		Platform:  strings.TrimSpace(c.Query("platform")),
		CycleUnit: strings.TrimSpace(c.Query("cycle_unit")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesStorePackageToResponse(&items[i]))
	}
	response.Paginated(c, out, pag.Total, page, pageSize)
}

func (h *SalesHandler) ListPartnerPackages(c *gin.Context) {
	partner, ok := h.authenticatePartner(c)
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, pag, err := h.salesService.ListPartnerCatalog(c.Request.Context(), partner.ID, pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesCatalogFilters{
		Platform:  strings.TrimSpace(c.Query("platform")),
		CycleUnit: strings.TrimSpace(c.Query("cycle_unit")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesPartnerCatalogToResponse(&items[i]))
	}
	response.Paginated(c, out, pag.Total, page, pageSize)
}

func (h *SalesHandler) ProvisionOrder(c *gin.Context) {
	partner, ok := h.authenticatePartner(c)
	if !ok {
		return
	}
	var req salesPartnerProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.ProvisionOrder(c.Request.Context(), &service.SalesProvisionInput{
		PartnerID:           partner.ID,
		ExternalOrderID:     req.ExternalOrderID,
		ExternalUserID:      req.ExternalUserID,
		ExternalEmail:       req.ExternalEmail,
		ExternalName:        req.ExternalName,
		APIKey:              req.APIKey,
		PackageID:           req.PackageID,
		ExternalPackageCode: req.ExternalPackageCode,
		OrderType:           req.OrderType,
		Amount:              req.Amount,
		Currency:            req.Currency,
		RawPayload:          req.RawPayload,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesProvisionCreatedResponse(c, item))
}

func (h *SalesHandler) GetPartnerOrder(c *gin.Context) {
	partner, ok := h.authenticatePartner(c)
	if !ok {
		return
	}
	externalOrderID := strings.TrimSpace(c.Param("external_order_id"))
	if externalOrderID == "" {
		response.BadRequest(c, "Invalid external_order_id")
		return
	}
	item, err := h.salesService.GetPartnerOrderResult(c.Request.Context(), partner.ID, externalOrderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesProvisionCreatedResponse(c, item))
}

func (h *SalesHandler) authenticatePartner(c *gin.Context) (*service.SalesPartner, bool) {
	secret := strings.TrimSpace(c.GetHeader(salesPartnerSecretHeader))
	if secret == "" {
		authorization := strings.TrimSpace(c.GetHeader(salesAuthorizationHeader))
		if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			secret = strings.TrimSpace(authorization[7:])
		}
	}
	if secret == "" {
		response.Unauthorized(c, "missing sales partner credentials")
		return nil, false
	}
	partner, err := h.salesService.AuthenticatePartnerBySecret(c.Request.Context(), secret)
	if err != nil {
		response.ErrorFrom(c, err)
		return nil, false
	}
	return partner, true
}

func salesCatalogToResponse(item *service.SalesCatalogItem) gin.H {
	if item == nil {
		return gin.H{}
	}
	return gin.H{
		"mapping_id":             item.MappingID,
		"partner_id":             item.PartnerID,
		"package_id":             item.PackageID,
		"external_package_code":  item.ExternalPackageCode,
		"external_package_name":  item.ExternalPackageName,
		"sale_price":             item.SalePrice,
		"currency":               item.Currency,
		"partner_package_status": item.PartnerPackageStatus,
		"package":                salesStorePackageToResponse(item.Package),
	}
}

func salesPartnerCatalogToResponse(item *service.SalesCatalogItem) gin.H {
	if item == nil {
		return gin.H{}
	}
	name := strings.TrimSpace(item.ExternalPackageName)
	if name == "" && item.Package != nil {
		name = strings.TrimSpace(item.Package.Name)
	}
	content := ""
	if item.Package != nil {
		if item.Package.Group != nil && strings.TrimSpace(item.Package.Group.Name) != "" {
			content = strings.TrimSpace(item.Package.Group.Name)
		} else if strings.TrimSpace(item.Package.Description) != "" {
			content = strings.TrimSpace(item.Package.Description)
		} else {
			content = strings.TrimSpace(item.Package.Name)
		}
	}
	packageType := ""
	if item.Package != nil {
		packageType = strings.TrimSpace(item.Package.CycleUnit)
	}
	return gin.H{
		"package_id":   item.PackageID,
		"name":         name,
		"package_type": packageType,
		"content":      content,
		"price":        item.SalePrice,
	}
}

func salesStorePackageToResponse(item *service.SalesPackage) gin.H {
	if item == nil {
		return gin.H{}
	}
	return gin.H{
		"id":              item.ID,
		"code":            item.Code,
		"name":            item.Name,
		"description":     item.Description,
		"platform":        item.Platform,
		"group_id":        item.GroupID,
		"group":           salesGroupSummaryToResponse(item.Group),
		"cycle_unit":      item.CycleUnit,
		"cycle_count":     item.CycleCount,
		"validity_days":   item.ValidityDays,
		"key_policy":      item.KeyPolicy,
		"auto_create_key": item.AutoCreateKey,
		"status":          item.Status,
		"store_visible":   item.StoreVisible,
		"sort_order":      item.SortOrder,
		"created_at":      item.CreatedAt,
		"updated_at":      item.UpdatedAt,
	}
}

func salesGroupSummaryToResponse(item *service.Group) gin.H {
	if item == nil {
		return nil
	}
	return gin.H{
		"id":                    item.ID,
		"name":                  item.Name,
		"description":           item.Description,
		"platform":              item.Platform,
		"status":                item.Status,
		"subscription_type":     item.SubscriptionType,
		"default_validity_days": item.DefaultValidityDays,
	}
}

func salesProvisionResultToResponse(item *service.SalesProvisionResult) gin.H {
	if item == nil {
		return gin.H{}
	}
	var subscription any
	if item.Subscription != nil {
		subscription = dto.UserSubscriptionFromService(item.Subscription)
	}
	return gin.H{
		"order":                        salesOrderSummaryToResponse(item.Order),
		"user":                         salesUserSummaryToResponse(item.User),
		"subscription":                 subscription,
		"api_key":                      salesAPIKeySummaryToResponse(item.APIKey),
		"api_key_value":                item.APIKeyValue,
		"created_user":                 item.CreatedUser,
		"created_binding":              item.CreatedBinding,
		"reused_api_key":               item.ReusedAPIKey,
		"subscription_already_applied": item.SubscriptionAlreadyApplied,
	}
}

func salesProvisionCreatedResponse(c *gin.Context, item *service.SalesProvisionResult) gin.H {
	if item == nil || item.Order == nil {
		return gin.H{}
	}
	return gin.H{
		"order_id":     strings.TrimSpace(item.Order.ExternalOrderID),
		"api_key":      strings.TrimSpace(item.APIKeyValue),
		"api_base_url": buildSalesGatewayBaseURL(c),
		"created_at":   item.Order.CreatedAt,
	}
}

func buildSalesGatewayBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return "/v1"
	}
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	if forwardedProto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.ToLower(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]))
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return "/v1"
	}
	return scheme + "://" + host + "/v1"
}

func salesOrderSummaryToResponse(item *service.SalesOrder) gin.H {
	if item == nil {
		return gin.H{}
	}
	return gin.H{
		"id":                item.ID,
		"partner_id":        item.PartnerID,
		"external_order_id": item.ExternalOrderID,
		"external_user_id":  item.ExternalUserID,
		"user_id":           item.UserID,
		"package_id":        item.PackageID,
		"order_type":        item.OrderType,
		"status":            item.Status,
		"subscription_id":   item.SubscriptionID,
		"api_key_id":        item.APIKeyID,
		"amount":            item.Amount,
		"currency":          item.Currency,
		"package_snapshot":  item.PackageSnapshot,
		"result_snapshot":   item.ResultSnapshot,
		"error_message":     item.ErrorMessage,
		"fulfilled_at":      item.FulfilledAt,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"package":           salesStorePackageToResponse(item.Package),
		"user":              salesUserSummaryToResponse(item.User),
	}
}

func salesUserSummaryToResponse(item *service.User) gin.H {
	if item == nil {
		return nil
	}
	return gin.H{
		"id":          item.ID,
		"email":       item.Email,
		"username":    item.Username,
		"role":        item.Role,
		"status":      item.Status,
		"balance":     item.Balance,
		"concurrency": item.Concurrency,
	}
}

func salesAPIKeySummaryToResponse(item *service.APIKey) gin.H {
	if item == nil {
		return nil
	}
	return gin.H{
		"id":         item.ID,
		"user_id":    item.UserID,
		"name":       item.Name,
		"group_id":   item.GroupID,
		"status":     item.Status,
		"expires_at": item.ExpiresAt,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	}
}
