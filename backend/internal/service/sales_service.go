package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SalesAuthModeHeader = "header"

	SalesCycleUnitDay   = "day"
	SalesCycleUnitMonth = "month"

	SalesKeyPolicyReuseCurrent  = "reuse_current"
	SalesKeyPolicyCreateIfMiss  = "create_if_missing"
	SalesKeyPolicyRotateOnRenew = "rotate_on_renew"

	SalesOrderTypePurchase = "purchase"
	SalesOrderTypeRenewal  = "renewal"
	SalesOrderTypeManual   = "manual"

	SalesOrderStatusPending             = "pending"
	SalesOrderStatusSubscriptionApplied = "subscription_applied"
	SalesOrderStatusFulfilled           = "fulfilled"
	SalesOrderStatusFailed              = "failed"
)

var (
	ErrSalesPartnerNotFound        = infraerrors.NotFound("SALES_PARTNER_NOT_FOUND", "sales partner not found")
	ErrSalesPackageNotFound        = infraerrors.NotFound("SALES_PACKAGE_NOT_FOUND", "sales package not found")
	ErrSalesPartnerPackageNotFound = infraerrors.NotFound("SALES_PARTNER_PACKAGE_NOT_FOUND", "partner package mapping not found")
	ErrSalesBindingNotFound        = infraerrors.NotFound("SALES_BINDING_NOT_FOUND", "sales user binding not found")
	ErrSalesOrderNotFound          = infraerrors.NotFound("SALES_ORDER_NOT_FOUND", "sales order not found")
	ErrSalesRenewalSourceNotFound  = infraerrors.NotFound("SALES_RENEWAL_SOURCE_NOT_FOUND", "renewal api key not found for current partner")
	ErrSalesPartnerExists          = infraerrors.Conflict("SALES_PARTNER_EXISTS", "sales partner already exists")
	ErrSalesPackageExists          = infraerrors.Conflict("SALES_PACKAGE_EXISTS", "sales package already exists")
	ErrSalesPartnerPackageExists   = infraerrors.Conflict("SALES_PARTNER_PACKAGE_EXISTS", "partner package mapping already exists")
	ErrSalesOrderExists            = infraerrors.Conflict("SALES_ORDER_EXISTS", "sales order already exists")
	ErrSalesInvalidSecret          = infraerrors.Unauthorized("SALES_INVALID_SECRET", "invalid partner credentials")
	ErrSalesPartnerDisabled        = infraerrors.Forbidden("SALES_PARTNER_DISABLED", "sales partner is disabled")
	ErrSalesPackageDisabled        = infraerrors.BadRequest("SALES_PACKAGE_DISABLED", "sales package is disabled")
	ErrSalesGroupInvalid           = infraerrors.BadRequest("SALES_GROUP_INVALID", "sales package group must be an active subscription group")
	ErrSalesOrderConflict          = infraerrors.Conflict("SALES_ORDER_CONFLICT", "existing order payload conflicts with current request")
	ErrSalesBindingConflict        = infraerrors.Conflict("SALES_BINDING_CONFLICT", "external user is already bound to another internal user")
	ErrSalesInvalidInput           = infraerrors.BadRequest("SALES_INVALID_INPUT", "invalid sales request")
)

type SalesPartner struct {
	ID           int64
	Code         string
	Name         string
	Description  string
	Status       string
	AuthMode     string
	SecretHash   string `json:"-"`
	SecretHint   string
	RateLimitRPM int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SalesPackage struct {
	ID            int64
	Code          string
	Name          string
	Description   string
	Platform      string
	GroupID       int64
	Group         *Group
	CycleUnit     string
	CycleCount    int
	ValidityDays  int
	KeyPolicy     string
	AutoCreateKey bool
	Status        string
	StoreVisible  bool
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SalesPartnerPackage struct {
	ID                  int64
	PartnerID           int64
	PackageID           int64
	ExternalPackageCode string
	ExternalPackageName string
	SalePrice           float64
	Currency            string
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Partner             *SalesPartner
	Package             *SalesPackage
}

type SalesUserBinding struct {
	ID             int64
	PartnerID      int64
	ExternalUserID string
	UserID         int64
	ExternalEmail  string
	ExternalName   string
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Partner        *SalesPartner
	User           *User
}

type SalesOrder struct {
	ID              int64
	PartnerID       int64
	ExternalOrderID string
	ExternalUserID  string
	UserID          *int64
	PackageID       int64
	OrderType       string
	Status          string
	SubscriptionID  *int64
	APIKeyID        *int64
	Amount          float64
	Currency        string
	PackageSnapshot map[string]any
	RawPayload      map[string]any
	ResultSnapshot  map[string]any
	ErrorMessage    string
	FulfilledAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Partner         *SalesPartner
	Package         *SalesPackage
	User            *User
}

type SalesCatalogItem struct {
	MappingID            int64
	PartnerID            int64
	PackageID            int64
	ExternalPackageCode  string
	ExternalPackageName  string
	SalePrice            float64
	Currency             string
	PartnerPackageStatus string
	Package              *SalesPackage
}

type SalesProvisionResult struct {
	Order                      *SalesOrder
	User                       *User
	Subscription               *UserSubscription
	APIKey                     *APIKey
	APIKeyValue                string
	CreatedUser                bool
	CreatedBinding             bool
	ReusedAPIKey               bool
	SubscriptionAlreadyApplied bool
}

type SalesPartnerListFilters struct {
	Status string
	Search string
}

type SalesPackageListFilters struct {
	Status       string
	Platform     string
	Search       string
	StoreVisible *bool
}

type SalesPartnerPackageListFilters struct {
	Status    string
	PartnerID *int64
	PackageID *int64
}

type SalesBindingListFilters struct {
	PartnerID *int64
	UserID    *int64
	Search    string
}

type SalesOrderListFilters struct {
	Status    string
	PartnerID *int64
	PackageID *int64
	UserID    *int64
	Search    string
}

type SalesCatalogFilters struct {
	Platform  string
	CycleUnit string
}

type CreateSalesPartnerInput struct {
	Code         string
	Name         string
	Description  string
	Status       string
	RateLimitRPM int
	Secret       string
}

type UpdateSalesPartnerInput struct {
	Name         *string
	Description  *string
	Status       *string
	RateLimitRPM *int
}

type CreateSalesPackageInput struct {
	Code          string
	Name          string
	Description   string
	GroupID       int64
	CycleUnit     string
	CycleCount    int
	ValidityDays  int
	KeyPolicy     string
	AutoCreateKey *bool
	Status        string
	StoreVisible  *bool
	SortOrder     *int
}

type UpdateSalesPackageInput struct {
	Name          *string
	Description   *string
	GroupID       *int64
	CycleUnit     *string
	CycleCount    *int
	ValidityDays  *int
	KeyPolicy     *string
	AutoCreateKey *bool
	Status        *string
	StoreVisible  *bool
	SortOrder     *int
}

type UpsertSalesPartnerPackageInput struct {
	ID                  int64
	PartnerID           int64
	PackageID           int64
	ExternalPackageCode string
	ExternalPackageName string
	SalePrice           float64
	Currency            string
	Status              string
}

type UpsertSalesBindingInput struct {
	ID             int64
	PartnerID      int64
	ExternalUserID string
	UserID         int64
	ExternalEmail  string
	ExternalName   string
	Metadata       map[string]any
}

type SalesProvisionInput struct {
	PartnerID           int64
	ExternalOrderID     string
	ExternalUserID      string
	ExternalEmail       string
	ExternalName        string
	APIKey              string
	PackageID           int64
	ExternalPackageCode string
	OrderType           string
	Amount              float64
	Currency            string
	RawPayload          map[string]any
}

type SalesRepository interface {
	CreatePartner(ctx context.Context, partner *SalesPartner) error
	UpdatePartner(ctx context.Context, partner *SalesPartner) error
	DeletePartner(ctx context.Context, id int64) error
	GetPartnerByID(ctx context.Context, id int64) (*SalesPartner, error)
	GetPartnerByCode(ctx context.Context, code string) (*SalesPartner, error)
	GetPartnerBySecretHash(ctx context.Context, secretHash string) (*SalesPartner, error)
	ListPartners(ctx context.Context, params pagination.PaginationParams, filters SalesPartnerListFilters) ([]SalesPartner, *pagination.PaginationResult, error)

	CreatePackage(ctx context.Context, pkg *SalesPackage) error
	UpdatePackage(ctx context.Context, pkg *SalesPackage) error
	DeletePackage(ctx context.Context, id int64) error
	GetPackageByID(ctx context.Context, id int64) (*SalesPackage, error)
	ListPackages(ctx context.Context, params pagination.PaginationParams, filters SalesPackageListFilters) ([]SalesPackage, *pagination.PaginationResult, error)

	UpsertPartnerPackage(ctx context.Context, mapping *SalesPartnerPackage) error
	DeletePartnerPackage(ctx context.Context, id int64) error
	GetPartnerPackageByID(ctx context.Context, id int64) (*SalesPartnerPackage, error)
	GetPartnerPackageByPartnerPackageID(ctx context.Context, partnerID, packageID int64) (*SalesPartnerPackage, error)
	GetPartnerPackageByExternalCode(ctx context.Context, partnerID int64, externalCode string) (*SalesPartnerPackage, error)
	ListPartnerPackages(ctx context.Context, params pagination.PaginationParams, filters SalesPartnerPackageListFilters) ([]SalesPartnerPackage, *pagination.PaginationResult, error)
	ListCatalogByPartner(ctx context.Context, partnerID int64, params pagination.PaginationParams, filters SalesCatalogFilters) ([]SalesCatalogItem, *pagination.PaginationResult, error)
	ListStoreCatalog(ctx context.Context, params pagination.PaginationParams, filters SalesCatalogFilters) ([]SalesPackage, *pagination.PaginationResult, error)

	GetBindingByID(ctx context.Context, id int64) (*SalesUserBinding, error)
	GetBindingByPartnerExternalUserID(ctx context.Context, partnerID int64, externalUserID string) (*SalesUserBinding, error)
	GetBindingByPartnerUserID(ctx context.Context, partnerID, userID int64) (*SalesUserBinding, error)
	ListBindings(ctx context.Context, params pagination.PaginationParams, filters SalesBindingListFilters) ([]SalesUserBinding, *pagination.PaginationResult, error)
	CreateBinding(ctx context.Context, binding *SalesUserBinding) error
	UpdateBinding(ctx context.Context, binding *SalesUserBinding) error
	DeleteBinding(ctx context.Context, id int64) error

	CreateOrder(ctx context.Context, order *SalesOrder) error
	UpdateOrder(ctx context.Context, order *SalesOrder) error
	DeleteOrder(ctx context.Context, id int64) error
	DeleteOrders(ctx context.Context, ids []int64) error
	GetOrderByID(ctx context.Context, id int64) (*SalesOrder, error)
	GetOrderByPartnerExternalID(ctx context.Context, partnerID int64, externalOrderID string) (*SalesOrder, error)
	GetLatestFulfilledOrderByPartnerAPIKeyID(ctx context.Context, partnerID, apiKeyID int64) (*SalesOrder, error)
	ListOrders(ctx context.Context, params pagination.PaginationParams, filters SalesOrderListFilters) ([]SalesOrder, *pagination.PaginationResult, error)
}

type SalesService struct {
	repo                SalesRepository
	userRepo            UserRepository
	groupRepo           GroupRepository
	userSubRepo         UserSubscriptionRepository
	apiKeyRepo          APIKeyRepository
	subscriptionService *SubscriptionService
	apiKeyService       *APIKeyService
}

func NewSalesService(
	repo SalesRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	userSubRepo UserSubscriptionRepository,
	apiKeyRepo APIKeyRepository,
	subscriptionService *SubscriptionService,
	apiKeyService *APIKeyService,
) *SalesService {
	return &SalesService{
		repo:                repo,
		userRepo:            userRepo,
		groupRepo:           groupRepo,
		userSubRepo:         userSubRepo,
		apiKeyRepo:          apiKeyRepo,
		subscriptionService: subscriptionService,
		apiKeyService:       apiKeyService,
	}
}

func (s *SalesService) ListPartners(ctx context.Context, params pagination.PaginationParams, filters SalesPartnerListFilters) ([]SalesPartner, *pagination.PaginationResult, error) {
	return s.repo.ListPartners(ctx, params, filters)
}

func (s *SalesService) GetPartner(ctx context.Context, id int64) (*SalesPartner, error) {
	return s.repo.GetPartnerByID(ctx, id)
}

func (s *SalesService) CreatePartner(ctx context.Context, input *CreateSalesPartnerInput) (*SalesPartner, string, error) {
	if input == nil {
		return nil, "", ErrSalesInvalidInput
	}
	code := normalizeSalesCode(input.Code)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, "", ErrSalesInvalidInput
	}
	status := normalizeStatus(input.Status)
	if status == "" {
		status = StatusActive
	}
	secret := strings.TrimSpace(input.Secret)
	if secret == "" {
		var err error
		secret, err = generateSalesSecret()
		if err != nil {
			return nil, "", err
		}
	}
	rpm := input.RateLimitRPM
	if rpm <= 0 {
		rpm = 60
	}
	partner := &SalesPartner{
		Name:         name,
		Description:  strings.TrimSpace(input.Description),
		Status:       status,
		AuthMode:     SalesAuthModeHeader,
		SecretHash:   hashSalesSecret(secret),
		SecretHint:   secretHint(secret),
		RateLimitRPM: rpm,
	}

	if code != "" {
		partner.Code = code
		if err := s.repo.CreatePartner(ctx, partner); err != nil {
			return nil, "", err
		}
		return partner, secret, nil
	}

	for i := 0; i < 5; i++ {
		generatedCode, err := generateSalesPartnerCode(name)
		if err != nil {
			return nil, "", err
		}
		partner.Code = generatedCode
		if err := s.repo.CreatePartner(ctx, partner); err != nil {
			if errors.Is(err, ErrSalesPartnerExists) {
				continue
			}
			return nil, "", err
		}
		return partner, secret, nil
	}

	return nil, "", ErrSalesPartnerExists
}

func (s *SalesService) UpdatePartner(ctx context.Context, id int64, input *UpdateSalesPartnerInput) (*SalesPartner, error) {
	if input == nil {
		return nil, ErrSalesInvalidInput
	}
	partner, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		partner.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		partner.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		status := normalizeStatus(*input.Status)
		if status == "" {
			return nil, ErrSalesInvalidInput
		}
		partner.Status = status
	}
	if input.RateLimitRPM != nil && *input.RateLimitRPM > 0 {
		partner.RateLimitRPM = *input.RateLimitRPM
	}
	if strings.TrimSpace(partner.Name) == "" {
		return nil, ErrSalesInvalidInput
	}
	if err := s.repo.UpdatePartner(ctx, partner); err != nil {
		return nil, err
	}
	return partner, nil
}

func (s *SalesService) RotatePartnerSecret(ctx context.Context, id int64) (*SalesPartner, string, error) {
	partner, err := s.repo.GetPartnerByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	secret, err := generateSalesSecret()
	if err != nil {
		return nil, "", err
	}
	partner.SecretHash = hashSalesSecret(secret)
	partner.SecretHint = secretHint(secret)
	if err := s.repo.UpdatePartner(ctx, partner); err != nil {
		return nil, "", err
	}
	return partner, secret, nil
}

func (s *SalesService) DeletePartner(ctx context.Context, id int64) error {
	return s.repo.DeletePartner(ctx, id)
}

func (s *SalesService) AuthenticatePartner(ctx context.Context, code, secret string) (*SalesPartner, error) {
	partner, err := s.repo.GetPartnerByCode(ctx, normalizeSalesCode(code))
	if err != nil {
		return nil, err
	}
	if partner.Status != StatusActive {
		return nil, ErrSalesPartnerDisabled
	}
	if subtle.ConstantTimeCompare([]byte(hashSalesSecret(strings.TrimSpace(secret))), []byte(partner.SecretHash)) != 1 {
		return nil, ErrSalesInvalidSecret
	}
	return partner, nil
}

func (s *SalesService) AuthenticatePartnerBySecret(ctx context.Context, secret string) (*SalesPartner, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrSalesInvalidSecret
	}
	partner, err := s.repo.GetPartnerBySecretHash(ctx, hashSalesSecret(secret))
	if err != nil {
		return nil, err
	}
	if partner.Status != StatusActive {
		return nil, ErrSalesPartnerDisabled
	}
	return partner, nil
}

func (s *SalesService) ListPackages(ctx context.Context, params pagination.PaginationParams, filters SalesPackageListFilters) ([]SalesPackage, *pagination.PaginationResult, error) {
	return s.repo.ListPackages(ctx, params, filters)
}

func (s *SalesService) GetPackage(ctx context.Context, id int64) (*SalesPackage, error) {
	return s.repo.GetPackageByID(ctx, id)
}

func (s *SalesService) CreatePackage(ctx context.Context, input *CreateSalesPackageInput) (*SalesPackage, error) {
	if input == nil {
		return nil, ErrSalesInvalidInput
	}
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if !group.IsActive() || !group.IsSubscriptionType() {
		return nil, ErrSalesGroupInvalid
	}
	keyPolicy := normalizeSalesKeyPolicy(input.KeyPolicy)
	if keyPolicy == "" {
		keyPolicy = SalesKeyPolicyReuseCurrent
	}
	cycleUnit := normalizeSalesCycleUnit(input.CycleUnit)
	if cycleUnit == "" {
		cycleUnit = SalesCycleUnitMonth
	}
	cycleCount := input.CycleCount
	if cycleCount <= 0 {
		cycleCount = 1
	}
	validityDays := normalizePackageValidityDays(input.ValidityDays, cycleUnit, cycleCount)
	status := normalizeStatus(input.Status)
	if status == "" {
		status = StatusActive
	}
	autoCreateKey := true
	if input.AutoCreateKey != nil {
		autoCreateKey = *input.AutoCreateKey
	}
	storeVisible := false
	if input.StoreVisible != nil {
		storeVisible = *input.StoreVisible
	}
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	pkg := &SalesPackage{
		Code:          normalizeSalesCode(input.Code),
		Name:          strings.TrimSpace(input.Name),
		Description:   strings.TrimSpace(input.Description),
		Platform:      group.Platform,
		GroupID:       group.ID,
		Group:         group,
		CycleUnit:     cycleUnit,
		CycleCount:    cycleCount,
		ValidityDays:  validityDays,
		KeyPolicy:     keyPolicy,
		AutoCreateKey: autoCreateKey,
		Status:        status,
		StoreVisible:  storeVisible,
		SortOrder:     sortOrder,
	}
	if pkg.Code == "" || pkg.Name == "" {
		return nil, ErrSalesInvalidInput
	}
	if err := s.repo.CreatePackage(ctx, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (s *SalesService) UpdatePackage(ctx context.Context, id int64, input *UpdateSalesPackageInput) (*SalesPackage, error) {
	if input == nil {
		return nil, ErrSalesInvalidInput
	}
	pkg, err := s.repo.GetPackageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		pkg.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		pkg.Description = strings.TrimSpace(*input.Description)
	}
	if input.GroupID != nil && *input.GroupID > 0 && *input.GroupID != pkg.GroupID {
		group, groupErr := s.groupRepo.GetByID(ctx, *input.GroupID)
		if groupErr != nil {
			return nil, groupErr
		}
		if !group.IsActive() || !group.IsSubscriptionType() {
			return nil, ErrSalesGroupInvalid
		}
		pkg.GroupID = group.ID
		pkg.Group = group
		pkg.Platform = group.Platform
	}
	if input.CycleUnit != nil {
		cycleUnit := normalizeSalesCycleUnit(*input.CycleUnit)
		if cycleUnit == "" {
			return nil, ErrSalesInvalidInput
		}
		pkg.CycleUnit = cycleUnit
	}
	if input.CycleCount != nil && *input.CycleCount > 0 {
		pkg.CycleCount = *input.CycleCount
	}
	if input.ValidityDays != nil {
		pkg.ValidityDays = normalizePackageValidityDays(*input.ValidityDays, pkg.CycleUnit, pkg.CycleCount)
	}
	if input.KeyPolicy != nil {
		keyPolicy := normalizeSalesKeyPolicy(*input.KeyPolicy)
		if keyPolicy == "" {
			return nil, ErrSalesInvalidInput
		}
		pkg.KeyPolicy = keyPolicy
	}
	if input.AutoCreateKey != nil {
		pkg.AutoCreateKey = *input.AutoCreateKey
	}
	if input.Status != nil {
		status := normalizeStatus(*input.Status)
		if status == "" {
			return nil, ErrSalesInvalidInput
		}
		pkg.Status = status
	}
	if input.StoreVisible != nil {
		pkg.StoreVisible = *input.StoreVisible
	}
	if input.SortOrder != nil {
		pkg.SortOrder = *input.SortOrder
	}
	if strings.TrimSpace(pkg.Name) == "" {
		return nil, ErrSalesInvalidInput
	}
	if err := s.repo.UpdatePackage(ctx, pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (s *SalesService) DeletePackage(ctx context.Context, id int64) error {
	return s.repo.DeletePackage(ctx, id)
}

func (s *SalesService) ListPartnerPackages(ctx context.Context, params pagination.PaginationParams, filters SalesPartnerPackageListFilters) ([]SalesPartnerPackage, *pagination.PaginationResult, error) {
	return s.repo.ListPartnerPackages(ctx, params, filters)
}

func (s *SalesService) UpsertPartnerPackage(ctx context.Context, input *UpsertSalesPartnerPackageInput) (*SalesPartnerPackage, error) {
	if input == nil {
		return nil, ErrSalesInvalidInput
	}
	partner, err := s.repo.GetPartnerByID(ctx, input.PartnerID)
	if err != nil {
		return nil, err
	}
	pkg, err := s.repo.GetPackageByID(ctx, input.PackageID)
	if err != nil {
		return nil, err
	}
	mapping := &SalesPartnerPackage{
		ID:                  input.ID,
		PartnerID:           partner.ID,
		PackageID:           pkg.ID,
		ExternalPackageCode: strings.TrimSpace(input.ExternalPackageCode),
		ExternalPackageName: strings.TrimSpace(input.ExternalPackageName),
		SalePrice:           input.SalePrice,
		Currency:            normalizeSalesCurrency(input.Currency),
		Status:              normalizeStatus(input.Status),
		Partner:             partner,
		Package:             pkg,
	}
	if mapping.ExternalPackageCode == "" {
		return nil, ErrSalesInvalidInput
	}
	if mapping.Status == "" {
		mapping.Status = StatusActive
	}
	if mapping.Currency == "" {
		mapping.Currency = "CNY"
	}
	if err := s.repo.UpsertPartnerPackage(ctx, mapping); err != nil {
		return nil, err
	}
	if mapping.ID <= 0 {
		return s.repo.GetPartnerPackageByExternalCode(ctx, mapping.PartnerID, mapping.ExternalPackageCode)
	}
	return s.repo.GetPartnerPackageByID(ctx, mapping.ID)
}

func (s *SalesService) DeletePartnerPackage(ctx context.Context, id int64) error {
	return s.repo.DeletePartnerPackage(ctx, id)
}

func (s *SalesService) ListPartnerCatalog(ctx context.Context, partnerID int64, params pagination.PaginationParams, filters SalesCatalogFilters) ([]SalesCatalogItem, *pagination.PaginationResult, error) {
	return s.repo.ListCatalogByPartner(ctx, partnerID, params, filters)
}

func (s *SalesService) ListStoreCatalog(ctx context.Context, params pagination.PaginationParams, filters SalesCatalogFilters) ([]SalesPackage, *pagination.PaginationResult, error) {
	return s.repo.ListStoreCatalog(ctx, params, filters)
}

func (s *SalesService) ListBindings(ctx context.Context, params pagination.PaginationParams, filters SalesBindingListFilters) ([]SalesUserBinding, *pagination.PaginationResult, error) {
	return s.repo.ListBindings(ctx, params, filters)
}

func (s *SalesService) GetBinding(ctx context.Context, id int64) (*SalesUserBinding, error) {
	return s.repo.GetBindingByID(ctx, id)
}

func (s *SalesService) UpsertBinding(ctx context.Context, input *UpsertSalesBindingInput) (*SalesUserBinding, error) {
	if input == nil {
		return nil, ErrSalesInvalidInput
	}

	partnerID := input.PartnerID
	userID := input.UserID
	externalUserID := strings.TrimSpace(input.ExternalUserID)
	if partnerID <= 0 || userID <= 0 || externalUserID == "" {
		return nil, ErrSalesInvalidInput
	}

	partner, err := s.repo.GetPartnerByID(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var current *SalesUserBinding
	if input.ID > 0 {
		current, err = s.repo.GetBindingByID(ctx, input.ID)
		if err != nil {
			return nil, err
		}
	}

	byExternal, err := s.repo.GetBindingByPartnerExternalUserID(ctx, partnerID, externalUserID)
	if err != nil && !errors.Is(err, ErrSalesBindingNotFound) {
		return nil, err
	}
	if errors.Is(err, ErrSalesBindingNotFound) {
		byExternal = nil
	}

	byUser, err := s.repo.GetBindingByPartnerUserID(ctx, partnerID, userID)
	if err != nil && !errors.Is(err, ErrSalesBindingNotFound) {
		return nil, err
	}
	if errors.Is(err, ErrSalesBindingNotFound) {
		byUser = nil
	}

	target := current
	if target == nil && byExternal != nil {
		target = byExternal
	}
	if target == nil && byUser != nil {
		target = byUser
	}

	if target != nil {
		if byExternal != nil && byExternal.ID != target.ID {
			return nil, ErrSalesBindingConflict
		}
		if byUser != nil && byUser.ID != target.ID {
			return nil, ErrSalesBindingConflict
		}

		target.PartnerID = partner.ID
		target.ExternalUserID = externalUserID
		target.UserID = user.ID
		target.ExternalEmail = strings.TrimSpace(input.ExternalEmail)
		target.ExternalName = strings.TrimSpace(input.ExternalName)
		if input.Metadata != nil {
			target.Metadata = cloneMap(input.Metadata)
		} else if len(target.Metadata) == 0 {
			target.Metadata = map[string]any{"source": "admin_sales_binding"}
		}
		if err := s.repo.UpdateBinding(ctx, target); err != nil {
			return nil, err
		}
		return s.repo.GetBindingByID(ctx, target.ID)
	}

	binding := &SalesUserBinding{
		PartnerID:      partner.ID,
		ExternalUserID: externalUserID,
		UserID:         user.ID,
		ExternalEmail:  strings.TrimSpace(input.ExternalEmail),
		ExternalName:   strings.TrimSpace(input.ExternalName),
		Metadata:       cloneMap(input.Metadata),
		Partner:        partner,
		User:           user,
	}
	if len(binding.Metadata) == 0 {
		binding.Metadata = map[string]any{"source": "admin_sales_binding"}
	}
	if err := s.repo.CreateBinding(ctx, binding); err != nil {
		return nil, err
	}
	return s.repo.GetBindingByID(ctx, binding.ID)
}

func (s *SalesService) DeleteBinding(ctx context.Context, id int64) error {
	return s.repo.DeleteBinding(ctx, id)
}

func (s *SalesService) ListOrders(ctx context.Context, params pagination.PaginationParams, filters SalesOrderListFilters) ([]SalesOrder, *pagination.PaginationResult, error) {
	return s.repo.ListOrders(ctx, params, filters)
}

func (s *SalesService) GetOrder(ctx context.Context, id int64) (*SalesOrder, error) {
	return s.repo.GetOrderByID(ctx, id)
}

func (s *SalesService) DeleteOrder(ctx context.Context, id int64) error {
	return s.repo.DeleteOrder(ctx, id)
}

func (s *SalesService) BatchDeleteOrders(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.DeleteOrders(ctx, ids)
}

func (s *SalesService) GetOrderResult(ctx context.Context, id int64) (*SalesProvisionResult, error) {
	order, err := s.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.hydrateProvisionResult(ctx, &SalesProvisionResult{Order: order})
}

func (s *SalesService) GetPartnerOrder(ctx context.Context, partnerID int64, externalOrderID string) (*SalesOrder, error) {
	return s.repo.GetOrderByPartnerExternalID(ctx, partnerID, strings.TrimSpace(externalOrderID))
}

func (s *SalesService) GetPartnerOrderResult(ctx context.Context, partnerID int64, externalOrderID string) (*SalesProvisionResult, error) {
	order, err := s.repo.GetOrderByPartnerExternalID(ctx, partnerID, strings.TrimSpace(externalOrderID))
	if err != nil {
		return nil, err
	}
	return s.hydrateProvisionResult(ctx, &SalesProvisionResult{Order: order})
}

func (s *SalesService) ProvisionOrder(ctx context.Context, input *SalesProvisionInput) (*SalesProvisionResult, error) {
	if input == nil {
		return nil, ErrSalesInvalidInput
	}
	input.ExternalOrderID = strings.TrimSpace(input.ExternalOrderID)
	input.ExternalUserID = strings.TrimSpace(input.ExternalUserID)
	input.ExternalEmail = strings.TrimSpace(input.ExternalEmail)
	input.ExternalName = strings.TrimSpace(input.ExternalName)
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.ExternalPackageCode = strings.TrimSpace(input.ExternalPackageCode)
	input.OrderType = strings.TrimSpace(input.OrderType)
	if input.PartnerID <= 0 || input.ExternalOrderID == "" {
		return nil, ErrSalesInvalidInput
	}
	partner, err := s.repo.GetPartnerByID(ctx, input.PartnerID)
	if err != nil {
		return nil, err
	}
	if partner.Status != StatusActive {
		return nil, ErrSalesPartnerDisabled
	}
	pkg, err := s.normalizeProvisionInput(ctx, partner, input)
	if err != nil {
		return nil, err
	}
	if pkg.Status != StatusActive {
		return nil, ErrSalesPackageDisabled
	}

	order, err := s.ensureProvisionOrder(ctx, partner, pkg, input)
	if err != nil {
		return nil, err
	}
	if existingConflict := s.validateExistingOrder(order, pkg, input); existingConflict != nil {
		return nil, existingConflict
	}

	result := &SalesProvisionResult{Order: order}
	if order.Status == SalesOrderStatusFulfilled {
		return s.hydrateProvisionResult(ctx, result)
	}

	user, createdUser, createdBinding, err := s.resolveProvisionUser(ctx, partner, order, input)
	if err != nil {
		_ = s.markOrderFailed(ctx, order, err)
		return nil, err
	}
	result.User = user
	result.CreatedUser = createdUser
	result.CreatedBinding = createdBinding
	userID := user.ID
	order.UserID = &userID

	sub, alreadyApplied, err := s.ensureProvisionSubscription(ctx, partner, pkg, user, order)
	if err != nil {
		_ = s.markOrderFailed(ctx, order, err)
		return nil, err
	}
	result.Subscription = sub
	result.SubscriptionAlreadyApplied = alreadyApplied

	apiKey, apiKeyValue, reusedAPIKey, err := s.ensureProvisionAPIKey(ctx, pkg, user, order)
	if err != nil {
		// 订阅已成功落地，但 Key 生成失败时保留在中间状态，允许幂等重试继续补 Key。
		order.Status = SalesOrderStatusSubscriptionApplied
		order.ErrorMessage = err.Error()
		_ = s.repo.UpdateOrder(ctx, order)
		return nil, err
	}
	result.APIKey = apiKey
	result.APIKeyValue = apiKeyValue
	result.ReusedAPIKey = reusedAPIKey

	order.Status = SalesOrderStatusFulfilled
	order.ErrorMessage = ""
	now := time.Now()
	order.FulfilledAt = &now
	order.ResultSnapshot = map[string]any{
		"user_id":         user.ID,
		"subscription_id": sub.ID,
		"expires_at":      sub.ExpiresAt.UTC().Format(time.RFC3339),
		"api_key_id":      nullableInt64(order.APIKeyID),
		"api_key_name": nullableString(func() string {
			if apiKey != nil {
				return apiKey.Name
			}
			return ""
		}()),
	}
	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}
	result.Order = order
	return result, nil
}

func (s *SalesService) normalizeProvisionInput(ctx context.Context, partner *SalesPartner, input *SalesProvisionInput) (*SalesPackage, error) {
	if input == nil || partner == nil {
		return nil, ErrSalesInvalidInput
	}
	if input.APIKey != "" {
		return s.normalizeRenewalProvisionInput(ctx, partner, input)
	}
	if input.PackageID <= 0 && input.ExternalPackageCode == "" {
		return nil, ErrSalesInvalidInput
	}
	if input.ExternalUserID == "" {
		input.ExternalUserID = generateSalesExternalUserID(partner, input.ExternalOrderID)
	}
	if input.OrderType == "" {
		input.OrderType = SalesOrderTypePurchase
	}
	return s.resolveProvisionPackage(ctx, input)
}

func (s *SalesService) normalizeRenewalProvisionInput(ctx context.Context, partner *SalesPartner, input *SalesProvisionInput) (*SalesPackage, error) {
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, input.APIKey)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return nil, ErrSalesRenewalSourceNotFound
		}
		return nil, err
	}
	sourceOrder, err := s.repo.GetLatestFulfilledOrderByPartnerAPIKeyID(ctx, partner.ID, apiKey.ID)
	if err != nil {
		if errors.Is(err, ErrSalesOrderNotFound) {
			return nil, ErrSalesRenewalSourceNotFound
		}
		return nil, err
	}
	if input.PackageID > 0 && input.PackageID != sourceOrder.PackageID {
		return nil, ErrSalesOrderConflict
	}
	if input.ExternalPackageCode != "" {
		if sourceOrder.Package == nil || !strings.EqualFold(strings.TrimSpace(sourceOrder.Package.Code), input.ExternalPackageCode) {
			return nil, ErrSalesOrderConflict
		}
	}
	input.PackageID = sourceOrder.PackageID
	input.ExternalPackageCode = ""
	if input.ExternalUserID == "" {
		input.ExternalUserID = strings.TrimSpace(sourceOrder.ExternalUserID)
	}
	if input.ExternalUserID == "" {
		input.ExternalUserID = generateSalesExternalUserID(partner, input.APIKey)
	}
	if input.OrderType == "" {
		input.OrderType = SalesOrderTypeRenewal
	}
	return s.resolveProvisionPackage(ctx, input)
}

func (s *SalesService) hydrateProvisionResult(ctx context.Context, result *SalesProvisionResult) (*SalesProvisionResult, error) {
	if result == nil || result.Order == nil {
		return nil, ErrSalesInvalidInput
	}
	order := result.Order
	if order.UserID != nil && result.User == nil {
		user, err := s.userRepo.GetByID(ctx, *order.UserID)
		if err == nil {
			result.User = user
		}
	}
	if order.SubscriptionID != nil && result.Subscription == nil {
		sub, err := s.userSubRepo.GetByID(ctx, *order.SubscriptionID)
		if err == nil {
			result.Subscription = sub
		}
	}
	if order.APIKeyID != nil && result.APIKey == nil {
		key, err := s.apiKeyRepo.GetByID(ctx, *order.APIKeyID)
		if err == nil {
			result.APIKey = key
			result.APIKeyValue = key.Key
		}
	}
	return result, nil
}

func (s *SalesService) resolveProvisionPackage(ctx context.Context, input *SalesProvisionInput) (*SalesPackage, error) {
	if input.PackageID > 0 {
		return s.repo.GetPackageByID(ctx, input.PackageID)
	}
	if input.ExternalPackageCode == "" {
		return nil, ErrSalesInvalidInput
	}
	mapping, err := s.repo.GetPartnerPackageByExternalCode(ctx, input.PartnerID, strings.TrimSpace(input.ExternalPackageCode))
	if err != nil {
		return nil, err
	}
	return mapping.Package, nil
}

func (s *SalesService) resolveConfiguredOrderAmount(ctx context.Context, partnerID, packageID int64) (float64, error) {
	mapping, err := s.repo.GetPartnerPackageByPartnerPackageID(ctx, partnerID, packageID)
	if err != nil {
		return 0, err
	}
	return mapping.SalePrice, nil
}

func (s *SalesService) ensureProvisionOrder(ctx context.Context, partner *SalesPartner, pkg *SalesPackage, input *SalesProvisionInput) (*SalesOrder, error) {
	order, err := s.repo.GetOrderByPartnerExternalID(ctx, partner.ID, input.ExternalOrderID)
	if err == nil {
		return order, nil
	}
	if !errors.Is(err, ErrSalesOrderNotFound) {
		return nil, err
	}

	orderType := strings.TrimSpace(input.OrderType)
	if orderType == "" {
		orderType = SalesOrderTypePurchase
	}
	amount, err := s.resolveConfiguredOrderAmount(ctx, partner.ID, pkg.ID)
	if err != nil {
		return nil, err
	}
	order = &SalesOrder{
		PartnerID:       partner.ID,
		ExternalOrderID: input.ExternalOrderID,
		ExternalUserID:  input.ExternalUserID,
		PackageID:       pkg.ID,
		OrderType:       orderType,
		Status:          SalesOrderStatusPending,
		Amount:          amount,
		Currency:        normalizeSalesCurrency(input.Currency),
		PackageSnapshot: buildPackageSnapshot(pkg),
		RawPayload:      cloneMap(input.RawPayload),
		ResultSnapshot:  map[string]any{},
	}
	if order.Currency == "" {
		order.Currency = "CNY"
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		if errors.Is(err, ErrSalesOrderExists) {
			return s.repo.GetOrderByPartnerExternalID(ctx, partner.ID, input.ExternalOrderID)
		}
		return nil, err
	}
	return order, nil
}

func (s *SalesService) validateExistingOrder(order *SalesOrder, pkg *SalesPackage, input *SalesProvisionInput) error {
	if order == nil {
		return nil
	}
	if order.PackageID != pkg.ID || order.ExternalUserID != input.ExternalUserID {
		return ErrSalesOrderConflict
	}
	return nil
}

func (s *SalesService) resolveProvisionUser(ctx context.Context, partner *SalesPartner, order *SalesOrder, input *SalesProvisionInput) (*User, bool, bool, error) {
	if order.UserID != nil && *order.UserID > 0 {
		user, err := s.userRepo.GetByID(ctx, *order.UserID)
		if err == nil {
			return user, false, false, nil
		}
	}

	if binding, err := s.repo.GetBindingByPartnerExternalUserID(ctx, partner.ID, input.ExternalUserID); err == nil {
		user, userErr := s.userRepo.GetByID(ctx, binding.UserID)
		if userErr == nil {
			return user, false, false, nil
		}
	}

	user, err := s.createProvisionUser(ctx, partner, input)
	if err != nil {
		return nil, false, false, err
	}
	createdUser := true

	createdBinding := false
	binding, err := s.repo.GetBindingByPartnerExternalUserID(ctx, partner.ID, input.ExternalUserID)
	if err != nil {
		if !errors.Is(err, ErrSalesBindingNotFound) {
			return nil, false, false, err
		}
		binding = &SalesUserBinding{
			PartnerID:      partner.ID,
			ExternalUserID: input.ExternalUserID,
			UserID:         user.ID,
			ExternalEmail:  input.ExternalEmail,
			ExternalName:   input.ExternalName,
			Metadata: map[string]any{
				"source": "sales_provision",
			},
		}
		if err := s.repo.CreateBinding(ctx, binding); err != nil {
			return nil, false, false, err
		}
		createdBinding = true
	} else if binding.UserID != user.ID {
		return nil, false, false, ErrSalesBindingConflict
	} else {
		if input.ExternalEmail != "" {
			binding.ExternalEmail = input.ExternalEmail
		}
		if input.ExternalName != "" {
			binding.ExternalName = input.ExternalName
		}
		if err := s.repo.UpdateBinding(ctx, binding); err != nil {
			return nil, false, false, err
		}
	}

	return user, createdUser, createdBinding, nil
}

func (s *SalesService) createProvisionUser(ctx context.Context, partner *SalesPartner, input *SalesProvisionInput) (*User, error) {
	password, err := generateSalesPassword()
	if err != nil {
		return nil, err
	}
	email := normalizeSalesEmail(partner, input)
	username := normalizeSalesUsername(partner, input)
	notes := fmt.Sprintf("%s %s for external_user_id=%s", SalesShadowUserNotesMarker, partner.Code, input.ExternalUserID)
	if input.ExternalEmail != "" {
		notes += "\nexternal_email=" + strings.ToLower(strings.TrimSpace(input.ExternalEmail))
	}
	if input.ExternalName != "" {
		notes += "\nexternal_name=" + strings.TrimSpace(input.ExternalName)
	}
	user := &User{
		Email:       email,
		Username:    username,
		Notes:       notes,
		Role:        RoleUser,
		Balance:     0,
		Concurrency: 5,
		Status:      StatusActive,
	}
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, ErrEmailExists) {
			if existing, getErr := s.userRepo.GetByEmail(ctx, email); getErr == nil {
				return existing, nil
			}
		}
		return nil, err
	}
	return user, nil
}

func (s *SalesService) ensureProvisionSubscription(ctx context.Context, partner *SalesPartner, pkg *SalesPackage, user *User, order *SalesOrder) (*UserSubscription, bool, error) {
	if order.SubscriptionID != nil && *order.SubscriptionID > 0 {
		sub, err := s.userSubRepo.GetByID(ctx, *order.SubscriptionID)
		if err == nil {
			return sub, true, nil
		}
	}

	marker := buildSalesOrderMarker(partner.Code, order.ExternalOrderID)
	existingSub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, user.ID, pkg.GroupID)
	if err == nil && existingSub != nil && strings.Contains(existingSub.Notes, marker) {
		order.SubscriptionID = &existingSub.ID
		order.Status = SalesOrderStatusSubscriptionApplied
		order.ErrorMessage = ""
		order.ResultSnapshot = map[string]any{
			"user_id":         user.ID,
			"subscription_id": existingSub.ID,
			"expires_at":      existingSub.ExpiresAt.UTC().Format(time.RFC3339),
		}
		_ = s.repo.UpdateOrder(ctx, order)
		return existingSub, true, nil
	}

	notes := buildSalesOrderNote(marker, pkg, partner, order)
	sub, _, err := s.subscriptionService.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
		UserID:       user.ID,
		GroupID:      pkg.GroupID,
		ValidityDays: pkg.ValidityDays,
		AssignedBy:   0,
		Notes:        notes,
	})
	if err != nil {
		return nil, false, err
	}
	order.SubscriptionID = &sub.ID
	order.Status = SalesOrderStatusSubscriptionApplied
	order.ErrorMessage = ""
	order.ResultSnapshot = map[string]any{
		"user_id":         user.ID,
		"subscription_id": sub.ID,
		"expires_at":      sub.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return nil, false, err
	}
	return sub, false, nil
}

func (s *SalesService) ensureProvisionAPIKey(ctx context.Context, pkg *SalesPackage, user *User, order *SalesOrder) (*APIKey, string, bool, error) {
	if !pkg.AutoCreateKey {
		return nil, "", true, nil
	}
	if order.APIKeyID != nil && *order.APIKeyID > 0 {
		key, err := s.apiKeyRepo.GetByID(ctx, *order.APIKeyID)
		if err == nil {
			return key, key.Key, true, nil
		}
	}

	reuseAllowed := pkg.KeyPolicy == SalesKeyPolicyReuseCurrent || pkg.KeyPolicy == SalesKeyPolicyCreateIfMiss || pkg.KeyPolicy == ""
	if reuseAllowed {
		groupID := pkg.GroupID
		keys, _, err := s.apiKeyRepo.ListByUserID(ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 200}, APIKeyListFilters{
			GroupID: &groupID,
			Status:  StatusActive,
		})
		if err == nil {
			for i := range keys {
				if keys[i].IsExpired() {
					continue
				}
				order.APIKeyID = &keys[i].ID
				return &keys[i], keys[i].Key, true, nil
			}
		}
	}

	groupID := pkg.GroupID
	key, err := s.apiKeyService.Create(ctx, user.ID, CreateAPIKeyRequest{
		Name:    buildSalesKeyName(pkg, order),
		GroupID: &groupID,
	})
	if err != nil {
		return nil, "", false, err
	}
	order.APIKeyID = &key.ID
	return key, key.Key, false, nil
}

func (s *SalesService) markOrderFailed(ctx context.Context, order *SalesOrder, cause error) error {
	if order == nil {
		return nil
	}
	order.Status = SalesOrderStatusFailed
	if cause != nil {
		order.ErrorMessage = cause.Error()
	}
	return s.repo.UpdateOrder(ctx, order)
}

func buildSalesOrderMarker(partnerCode, externalOrderID string) string {
	return "[sales_order:" + normalizeSalesCode(partnerCode) + ":" + strings.TrimSpace(externalOrderID) + "]"
}

func buildSalesOrderNote(marker string, pkg *SalesPackage, partner *SalesPartner, order *SalesOrder) string {
	parts := []string{
		marker,
		fmt.Sprintf("partner=%s", partner.Code),
		fmt.Sprintf("package=%s", pkg.Code),
		fmt.Sprintf("external_order_id=%s", order.ExternalOrderID),
	}
	return strings.Join(parts, "\n")
}

func buildPackageSnapshot(pkg *SalesPackage) map[string]any {
	if pkg == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":              pkg.ID,
		"code":            pkg.Code,
		"name":            pkg.Name,
		"platform":        pkg.Platform,
		"group_id":        pkg.GroupID,
		"cycle_unit":      pkg.CycleUnit,
		"cycle_count":     pkg.CycleCount,
		"validity_days":   pkg.ValidityDays,
		"key_policy":      pkg.KeyPolicy,
		"auto_create_key": pkg.AutoCreateKey,
	}
}

func buildSalesKeyName(pkg *SalesPackage, order *SalesOrder) string {
	base := pkg.Name
	if strings.TrimSpace(base) == "" {
		base = pkg.Code
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(base), strings.TrimSpace(order.ExternalOrderID))
}

func generateSalesSecret() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func generateSalesPassword() (string, error) {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func hashSalesSecret(secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return hex.EncodeToString(sum[:])
}

func secretHint(secret string) string {
	secret = strings.TrimSpace(secret)
	if len(secret) <= 6 {
		return secret
	}
	return secret[len(secret)-6:]
}

func generateSalesPartnerCode(name string) (string, error) {
	suffixBytes := make([]byte, 4)
	if _, err := rand.Read(suffixBytes); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(suffixBytes)
	prefix := sanitizeSalesSlug(name)
	if prefix == "" {
		prefix = "partner"
	}
	maxPrefixLen := 64 - len(suffix) - 1
	if maxPrefixLen < 1 {
		maxPrefixLen = 1
	}
	if len(prefix) > maxPrefixLen {
		prefix = prefix[:maxPrefixLen]
	}
	return prefix + "-" + suffix, nil
}

func normalizeSalesCode(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))
	code = strings.ReplaceAll(code, " ", "-")
	return code
}

func normalizeSalesCurrency(currency string) string {
	currency = strings.TrimSpace(strings.ToUpper(currency))
	if currency == "" {
		return ""
	}
	return currency
}

func normalizeSalesCycleUnit(unit string) string {
	switch strings.TrimSpace(strings.ToLower(unit)) {
	case SalesCycleUnitDay:
		return SalesCycleUnitDay
	case SalesCycleUnitMonth:
		return SalesCycleUnitMonth
	default:
		return ""
	}
}

func normalizeSalesKeyPolicy(policy string) string {
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case SalesKeyPolicyReuseCurrent:
		return SalesKeyPolicyReuseCurrent
	case SalesKeyPolicyCreateIfMiss:
		return SalesKeyPolicyCreateIfMiss
	case SalesKeyPolicyRotateOnRenew:
		return SalesKeyPolicyRotateOnRenew
	default:
		return ""
	}
}

func normalizePackageValidityDays(validityDays int, cycleUnit string, cycleCount int) int {
	if validityDays > 0 {
		if validityDays > MaxValidityDays {
			return MaxValidityDays
		}
		return validityDays
	}
	if cycleCount <= 0 {
		cycleCount = 1
	}
	if cycleUnit == SalesCycleUnitDay {
		return cycleCount
	}
	return 30 * cycleCount
}

func normalizeSalesEmail(partner *SalesPartner, input *SalesProvisionInput) string {
	partnerCode := sanitizeSalesSlug(partner.Code)
	if partnerCode == "" {
		partnerCode = "partner"
	}
	if len(partnerCode) > 24 {
		partnerCode = partnerCode[:24]
	}
	base := sanitizeSalesSlug(input.ExternalUserID)
	if base == "" {
		base = "salesuser"
	}
	if len(base) > 24 {
		base = base[:24]
	}
	sum := sha256.Sum256([]byte(normalizeSalesCode(partner.Code) + ":" + strings.TrimSpace(input.ExternalUserID)))
	return fmt.Sprintf("%s_%s_%s%s", partnerCode, base, hex.EncodeToString(sum[:6]), SalesShadowUserEmailSuffix)
}

func generateSalesExternalUserID(partner *SalesPartner, seed string) string {
	partnerCode := sanitizeSalesSlug(partner.Code)
	if partnerCode == "" {
		partnerCode = "partner"
	}
	if len(partnerCode) > 24 {
		partnerCode = partnerCode[:24]
	}
	sum := sha256.Sum256([]byte(normalizeSalesCode(partner.Code) + ":" + strings.TrimSpace(seed)))
	return fmt.Sprintf("sales_%s_%s", partnerCode, hex.EncodeToString(sum[:6]))
}

func normalizeSalesUsername(partner *SalesPartner, input *SalesProvisionInput) string {
	partnerCode := sanitizeSalesSlug(partner.Code)
	if partnerCode == "" {
		partnerCode = "partner"
	}
	if len(partnerCode) > 24 {
		partnerCode = partnerCode[:24]
	}
	base := input.ExternalName
	if strings.TrimSpace(base) == "" {
		base = input.ExternalUserID
	}
	base = sanitizeSalesSlug(base)
	if base == "" {
		base = "salesuser"
	}
	if len(base) > 32 {
		base = base[:32]
	}
	sum := sha256.Sum256([]byte(normalizeSalesCode(partner.Code) + ":" + strings.TrimSpace(input.ExternalUserID)))
	return fmt.Sprintf("%s_%s_%s", partnerCode, base, hex.EncodeToString(sum[:4]))
}

func sanitizeSalesSlug(input string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(input)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-_")
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func normalizeStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case StatusActive:
		return StatusActive
	case StatusDisabled:
		return StatusDisabled
	default:
		return ""
	}
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
