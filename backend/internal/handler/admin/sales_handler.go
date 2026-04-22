package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type SalesHandler struct {
	salesService *service.SalesService
}

func NewSalesHandler(salesService *service.SalesService) *SalesHandler {
	return &SalesHandler{salesService: salesService}
}

type createSalesPartnerRequest struct {
	Code         string `json:"code" binding:"omitempty,max=64"`
	Name         string `json:"name" binding:"required,max=100"`
	Description  string `json:"description"`
	Status       string `json:"status" binding:"omitempty,oneof=active disabled"`
	RateLimitRPM int    `json:"rate_limit_rpm" binding:"omitempty,min=1,max=100000"`
	Secret       string `json:"secret"`
}

type updateSalesPartnerRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	Status       *string `json:"status" binding:"omitempty,oneof=active disabled"`
	RateLimitRPM *int    `json:"rate_limit_rpm" binding:"omitempty,min=1,max=100000"`
}

type createSalesPackageRequest struct {
	Code          string `json:"code" binding:"required,max=64"`
	Name          string `json:"name" binding:"required,max=100"`
	Description   string `json:"description"`
	GroupID       int64  `json:"group_id" binding:"required"`
	CycleUnit     string `json:"cycle_unit" binding:"omitempty,oneof=day month"`
	CycleCount    int    `json:"cycle_count" binding:"omitempty,min=1,max=36500"`
	ValidityDays  int    `json:"validity_days" binding:"omitempty,min=1,max=36500"`
	KeyPolicy     string `json:"key_policy" binding:"omitempty,oneof=reuse_current create_if_missing rotate_on_renew"`
	AutoCreateKey *bool  `json:"auto_create_key"`
	Status        string `json:"status" binding:"omitempty,oneof=active disabled"`
	StoreVisible  *bool  `json:"store_visible"`
	SortOrder     *int   `json:"sort_order"`
}

type updateSalesPackageRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	GroupID       *int64  `json:"group_id"`
	CycleUnit     *string `json:"cycle_unit" binding:"omitempty,oneof=day month"`
	CycleCount    *int    `json:"cycle_count" binding:"omitempty,min=1,max=36500"`
	ValidityDays  *int    `json:"validity_days" binding:"omitempty,min=1,max=36500"`
	KeyPolicy     *string `json:"key_policy" binding:"omitempty,oneof=reuse_current create_if_missing rotate_on_renew"`
	AutoCreateKey *bool   `json:"auto_create_key"`
	Status        *string `json:"status" binding:"omitempty,oneof=active disabled"`
	StoreVisible  *bool   `json:"store_visible"`
	SortOrder     *int    `json:"sort_order"`
}

type upsertSalesPartnerPackageRequest struct {
	ID                  int64   `json:"id"`
	PartnerID           int64   `json:"partner_id" binding:"required"`
	PackageID           int64   `json:"package_id" binding:"required"`
	ExternalPackageCode string  `json:"external_package_code" binding:"required,max=100"`
	ExternalPackageName string  `json:"external_package_name"`
	SalePrice           float64 `json:"sale_price" binding:"omitempty,min=0"`
	Currency            string  `json:"currency" binding:"omitempty,max=16"`
	Status              string  `json:"status" binding:"omitempty,oneof=active disabled"`
}

type upsertSalesBindingRequest struct {
	ID             int64          `json:"id"`
	PartnerID      int64          `json:"partner_id" binding:"required"`
	ExternalUserID string         `json:"external_user_id" binding:"required,max=100"`
	UserID         int64          `json:"user_id" binding:"required"`
	ExternalEmail  string         `json:"external_email" binding:"omitempty,email"`
	ExternalName   string         `json:"external_name"`
	Metadata       map[string]any `json:"metadata"`
}

type salesProvisionRequest struct {
	PartnerID           int64          `json:"partner_id" binding:"required"`
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

func (h *SalesHandler) ListPartners(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, pag, err := h.salesService.ListPartners(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesPartnerListFilters{
		Status: strings.TrimSpace(c.Query("status")),
		Search: strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesPartnerToResponse(&items[i]))
	}
	response.PaginatedWithResult(c, out, toResponsePagination(pag))
}

func (h *SalesHandler) GetPartner(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.salesService.GetPartner(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesPartnerToResponse(item))
}

func (h *SalesHandler) CreatePartner(c *gin.Context) {
	var req createSalesPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, secret, err := h.salesService.CreatePartner(c.Request.Context(), &service.CreateSalesPartnerInput{
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		Status:       req.Status,
		RateLimitRPM: req.RateLimitRPM,
		Secret:       req.Secret,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, gin.H{
		"partner": salesPartnerToResponse(item),
		"secret":  secret,
	})
}

func (h *SalesHandler) UpdatePartner(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	var req updateSalesPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.UpdatePartner(c.Request.Context(), id, &service.UpdateSalesPartnerInput{
		Name:         req.Name,
		Description:  req.Description,
		Status:       req.Status,
		RateLimitRPM: req.RateLimitRPM,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesPartnerToResponse(item))
}

func (h *SalesHandler) RotatePartnerSecret(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	item, secret, err := h.salesService.RotatePartnerSecret(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"partner": salesPartnerToResponse(item),
		"secret":  secret,
	})
}

func (h *SalesHandler) DeletePartner(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.salesService.DeletePartner(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *SalesHandler) ListPackages(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var storeVisible *bool
	if raw := strings.TrimSpace(c.Query("store_visible")); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			storeVisible = &v
		}
	}
	items, pag, err := h.salesService.ListPackages(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesPackageListFilters{
		Status:       strings.TrimSpace(c.Query("status")),
		Platform:     strings.TrimSpace(c.Query("platform")),
		Search:       strings.TrimSpace(c.Query("search")),
		StoreVisible: storeVisible,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesPackageToResponse(&items[i]))
	}
	response.PaginatedWithResult(c, out, toResponsePagination(pag))
}

func (h *SalesHandler) GetPackage(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.salesService.GetPackage(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesPackageToResponse(item))
}

func (h *SalesHandler) CreatePackage(c *gin.Context) {
	var req createSalesPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.CreatePackage(c.Request.Context(), &service.CreateSalesPackageInput{
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		GroupID:       req.GroupID,
		CycleUnit:     req.CycleUnit,
		CycleCount:    req.CycleCount,
		ValidityDays:  req.ValidityDays,
		KeyPolicy:     req.KeyPolicy,
		AutoCreateKey: req.AutoCreateKey,
		Status:        req.Status,
		StoreVisible:  req.StoreVisible,
		SortOrder:     req.SortOrder,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, salesPackageToResponse(item))
}

func (h *SalesHandler) UpdatePackage(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	var req updateSalesPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.UpdatePackage(c.Request.Context(), id, &service.UpdateSalesPackageInput{
		Name:          req.Name,
		Description:   req.Description,
		GroupID:       req.GroupID,
		CycleUnit:     req.CycleUnit,
		CycleCount:    req.CycleCount,
		ValidityDays:  req.ValidityDays,
		KeyPolicy:     req.KeyPolicy,
		AutoCreateKey: req.AutoCreateKey,
		Status:        req.Status,
		StoreVisible:  req.StoreVisible,
		SortOrder:     req.SortOrder,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesPackageToResponse(item))
}

func (h *SalesHandler) DeletePackage(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.salesService.DeletePackage(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *SalesHandler) ListMappings(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var partnerID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "partner_id"); ok {
		partnerID = &v
	}
	var packageID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "package_id"); ok {
		packageID = &v
	}
	items, pag, err := h.salesService.ListPartnerPackages(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesPartnerPackageListFilters{
		Status:    strings.TrimSpace(c.Query("status")),
		PartnerID: partnerID,
		PackageID: packageID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesMappingToResponse(&items[i]))
	}
	response.PaginatedWithResult(c, out, toResponsePagination(pag))
}

func (h *SalesHandler) UpsertMapping(c *gin.Context) {
	var req upsertSalesPartnerPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.UpsertPartnerPackage(c.Request.Context(), &service.UpsertSalesPartnerPackageInput{
		ID:                  req.ID,
		PartnerID:           req.PartnerID,
		PackageID:           req.PackageID,
		ExternalPackageCode: req.ExternalPackageCode,
		ExternalPackageName: req.ExternalPackageName,
		SalePrice:           req.SalePrice,
		Currency:            req.Currency,
		Status:              req.Status,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesMappingToResponse(item))
}

func (h *SalesHandler) DeleteMapping(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.salesService.DeletePartnerPackage(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *SalesHandler) ListBindings(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var partnerID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "partner_id"); ok {
		partnerID = &v
	}
	var userID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "user_id"); ok {
		userID = &v
	}
	items, pag, err := h.salesService.ListBindings(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesBindingListFilters{
		PartnerID: partnerID,
		UserID:    userID,
		Search:    strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesBindingToResponse(&items[i]))
	}
	response.PaginatedWithResult(c, out, toResponsePagination(pag))
}

func (h *SalesHandler) GetBinding(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.salesService.GetBinding(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesBindingToResponse(item))
}

func (h *SalesHandler) UpsertBinding(c *gin.Context) {
	var req upsertSalesBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.UpsertBinding(c.Request.Context(), &service.UpsertSalesBindingInput{
		ID:             req.ID,
		PartnerID:      req.PartnerID,
		ExternalUserID: req.ExternalUserID,
		UserID:         req.UserID,
		ExternalEmail:  req.ExternalEmail,
		ExternalName:   req.ExternalName,
		Metadata:       req.Metadata,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesBindingToResponse(item))
}

func (h *SalesHandler) DeleteBinding(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.salesService.DeleteBinding(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *SalesHandler) ListOrders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	var partnerID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "partner_id"); ok {
		partnerID = &v
	}
	var packageID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "package_id"); ok {
		packageID = &v
	}
	var userID *int64
	if v, ok := parseSalesOptionalQueryInt64(c, "user_id"); ok {
		userID = &v
	}
	items, pag, err := h.salesService.ListOrders(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.SalesOrderListFilters{
		Status:    strings.TrimSpace(c.Query("status")),
		PartnerID: partnerID,
		PackageID: packageID,
		UserID:    userID,
		Search:    strings.TrimSpace(c.Query("search")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, salesOrderToResponse(&items[i]))
	}
	response.PaginatedWithResult(c, out, toResponsePagination(pag))
}

func (h *SalesHandler) GetOrder(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.salesService.GetOrderResult(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, salesProvisionToResponse(item))
}

func (h *SalesHandler) DeleteOrder(c *gin.Context) {
	id, ok := parseSalesIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.salesService.DeleteOrder(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *SalesHandler) BatchDeleteOrders(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required,dive,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		response.BadRequest(c, "Invalid request: ids is required")
		return
	}
	if err := h.salesService.BatchDeleteOrders(c.Request.Context(), req.IDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": len(req.IDs)})
}

func (h *SalesHandler) ProvisionOrder(c *gin.Context) {
	var req salesProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.salesService.ProvisionOrder(c.Request.Context(), &service.SalesProvisionInput{
		PartnerID:           req.PartnerID,
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
	response.Success(c, salesProvisionToResponse(item))
}

func parseSalesIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return id, true
}

func parseSalesOptionalQueryInt64(c *gin.Context, name string) (int64, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func salesPartnerToResponse(item *service.SalesPartner) gin.H {
	if item == nil {
		return gin.H{}
	}
	return gin.H{
		"id":             item.ID,
		"code":           item.Code,
		"name":           item.Name,
		"description":    item.Description,
		"status":         item.Status,
		"auth_mode":      item.AuthMode,
		"secret_hint":    item.SecretHint,
		"rate_limit_rpm": item.RateLimitRPM,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func salesPackageToResponse(item *service.SalesPackage) gin.H {
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
		"group":           salesGroupToResponse(item.Group),
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

func salesGroupToResponse(item *service.Group) gin.H {
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

func salesMappingToResponse(item *service.SalesPartnerPackage) gin.H {
	if item == nil {
		return gin.H{}
	}
	return gin.H{
		"id":                    item.ID,
		"partner_id":            item.PartnerID,
		"package_id":            item.PackageID,
		"external_package_code": item.ExternalPackageCode,
		"external_package_name": item.ExternalPackageName,
		"sale_price":            item.SalePrice,
		"currency":              item.Currency,
		"status":                item.Status,
		"created_at":            item.CreatedAt,
		"updated_at":            item.UpdatedAt,
		"partner":               salesPartnerToResponse(item.Partner),
		"package":               salesPackageToResponse(item.Package),
	}
}

func salesBindingToResponse(item *service.SalesUserBinding) gin.H {
	if item == nil {
		return gin.H{}
	}
	return gin.H{
		"id":               item.ID,
		"partner_id":       item.PartnerID,
		"external_user_id": item.ExternalUserID,
		"user_id":          item.UserID,
		"external_email":   item.ExternalEmail,
		"external_name":    item.ExternalName,
		"metadata":         item.Metadata,
		"created_at":       item.CreatedAt,
		"updated_at":       item.UpdatedAt,
		"partner":          salesPartnerToResponse(item.Partner),
		"user":             salesUserToResponse(item.User),
	}
}

func salesOrderToResponse(item *service.SalesOrder) gin.H {
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
		"raw_payload":       item.RawPayload,
		"result_snapshot":   item.ResultSnapshot,
		"error_message":     item.ErrorMessage,
		"fulfilled_at":      item.FulfilledAt,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
		"partner":           salesPartnerToResponse(item.Partner),
		"package":           salesPackageToResponse(item.Package),
		"user":              salesUserToResponse(item.User),
	}
}

func salesProvisionToResponse(item *service.SalesProvisionResult) gin.H {
	if item == nil {
		return gin.H{}
	}
	var subscription any
	if item.Subscription != nil {
		subscription = dto.UserSubscriptionFromServiceAdmin(item.Subscription)
	}
	return gin.H{
		"order":                        salesOrderToResponse(item.Order),
		"user":                         salesUserToResponse(item.User),
		"subscription":                 subscription,
		"api_key":                      salesAPIKeyToResponse(item.APIKey),
		"api_key_value":                item.APIKeyValue,
		"created_user":                 item.CreatedUser,
		"created_binding":              item.CreatedBinding,
		"reused_api_key":               item.ReusedAPIKey,
		"subscription_already_applied": item.SubscriptionAlreadyApplied,
	}
}

func salesUserToResponse(item *service.User) gin.H {
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

func salesAPIKeyToResponse(item *service.APIKey) gin.H {
	if item == nil {
		return nil
	}
	return gin.H{
		"id":         item.ID,
		"user_id":    item.UserID,
		"name":       item.Name,
		"key":        item.Key,
		"group_id":   item.GroupID,
		"status":     item.Status,
		"expires_at": item.ExpiresAt,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	}
}
