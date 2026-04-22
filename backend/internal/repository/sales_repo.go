package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type salesRepository struct {
	db *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

func NewSalesRepository(db *sql.DB) service.SalesRepository {
	return &salesRepository{db: db}
}

func (r *salesRepository) runInTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *salesRepository) CreatePartner(ctx context.Context, partner *service.SalesPartner) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO sales_partners
		 (code, name, description, status, auth_mode, secret_hash, secret_hint, rate_limit_rpm)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		partner.Code, partner.Name, partner.Description, partner.Status, partner.AuthMode,
		partner.SecretHash, partner.SecretHint, partner.RateLimitRPM,
	).Scan(&partner.ID, &partner.CreatedAt, &partner.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesPartnerExists.WithCause(err)
		}
		return fmt.Errorf("insert sales partner: %w", err)
	}
	return nil
}

func (r *salesRepository) UpdatePartner(ctx context.Context, partner *service.SalesPartner) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE sales_partners
		 SET name = $1, description = $2, status = $3, auth_mode = $4,
		     secret_hash = $5, secret_hint = $6, rate_limit_rpm = $7, updated_at = NOW()
		 WHERE id = $8`,
		partner.Name, partner.Description, partner.Status, partner.AuthMode,
		partner.SecretHash, partner.SecretHint, partner.RateLimitRPM, partner.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesPartnerExists.WithCause(err)
		}
		return fmt.Errorf("update sales partner: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesPartnerNotFound
	}
	current, err := r.GetPartnerByID(ctx, partner.ID)
	if err != nil {
		return err
	}
	*partner = *current
	return nil
}

func (r *salesRepository) DeletePartner(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sales_partners WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sales partner: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesPartnerNotFound
	}
	return nil
}

func (r *salesRepository) GetPartnerByID(ctx context.Context, id int64) (*service.SalesPartner, error) {
	partner, err := scanSalesPartner(
		r.db.QueryRowContext(ctx,
			`SELECT id, code, name, description, status, auth_mode, secret_hash, secret_hint, rate_limit_rpm, created_at, updated_at
			 FROM sales_partners
			 WHERE id = $1`,
			id,
		),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPartnerNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales partner by id: %w", err)
	}
	return partner, nil
}

func (r *salesRepository) GetPartnerByCode(ctx context.Context, code string) (*service.SalesPartner, error) {
	partner, err := scanSalesPartner(
		r.db.QueryRowContext(ctx,
			`SELECT id, code, name, description, status, auth_mode, secret_hash, secret_hint, rate_limit_rpm, created_at, updated_at
			 FROM sales_partners
			 WHERE code = $1`,
			code,
		),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPartnerNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales partner by code: %w", err)
	}
	return partner, nil
}

func (r *salesRepository) GetPartnerBySecretHash(ctx context.Context, secretHash string) (*service.SalesPartner, error) {
	partner, err := scanSalesPartner(
		r.db.QueryRowContext(ctx,
			`SELECT id, code, name, description, status, auth_mode, secret_hash, secret_hint, rate_limit_rpm, created_at, updated_at
			 FROM sales_partners
			 WHERE secret_hash = $1
			 ORDER BY id DESC
			 LIMIT 1`,
			secretHash,
		),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPartnerNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales partner by secret hash: %w", err)
	}
	return partner, nil
}

func (r *salesRepository) ListPartners(ctx context.Context, params pagination.PaginationParams, filters service.SalesPartnerListFilters) ([]service.SalesPartner, *pagination.PaginationResult, error) {
	where, args := buildSalesPartnerWhere(filters)
	countQuery := `SELECT COUNT(*) FROM sales_partners sp WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count sales partners: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, code, name, description, status, auth_mode, secret_hash, secret_hint, rate_limit_rpm, created_at, updated_at
		 FROM sales_partners sp
		 WHERE `+where+`
		 ORDER BY sp.id DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query sales partners: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesPartner, 0)
	for rows.Next() {
		item, scanErr := scanSalesPartner(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan sales partner: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sales partners: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *salesRepository) CreatePackage(ctx context.Context, pkg *service.SalesPackage) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO sales_packages
		 (code, name, description, platform, group_id, cycle_unit, cycle_count, validity_days, key_policy, auto_create_key, status, store_visible, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING id, created_at, updated_at`,
		pkg.Code, pkg.Name, pkg.Description, pkg.Platform, pkg.GroupID, pkg.CycleUnit, pkg.CycleCount,
		pkg.ValidityDays, pkg.KeyPolicy, pkg.AutoCreateKey, pkg.Status, pkg.StoreVisible, pkg.SortOrder,
	).Scan(&pkg.ID, &pkg.CreatedAt, &pkg.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesPackageExists.WithCause(err)
		}
		return fmt.Errorf("insert sales package: %w", err)
	}
	return nil
}

func (r *salesRepository) UpdatePackage(ctx context.Context, pkg *service.SalesPackage) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE sales_packages
		 SET name = $1, description = $2, platform = $3, group_id = $4, cycle_unit = $5,
		     cycle_count = $6, validity_days = $7, key_policy = $8, auto_create_key = $9,
		     status = $10, store_visible = $11, sort_order = $12, updated_at = NOW()
		 WHERE id = $13`,
		pkg.Name, pkg.Description, pkg.Platform, pkg.GroupID, pkg.CycleUnit,
		pkg.CycleCount, pkg.ValidityDays, pkg.KeyPolicy, pkg.AutoCreateKey,
		pkg.Status, pkg.StoreVisible, pkg.SortOrder, pkg.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesPackageExists.WithCause(err)
		}
		return fmt.Errorf("update sales package: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesPackageNotFound
	}
	current, err := r.GetPackageByID(ctx, pkg.ID)
	if err != nil {
		return err
	}
	*pkg = *current
	return nil
}

func (r *salesRepository) DeletePackage(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sales_packages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sales package: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesPackageNotFound
	}
	return nil
}

func (r *salesRepository) GetPackageByID(ctx context.Context, id int64) (*service.SalesPackage, error) {
	pkg, err := scanSalesPackage(
		r.db.QueryRowContext(ctx, salesPackageSelect+` WHERE spkg.id = $1`, id),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPackageNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales package by id: %w", err)
	}
	return pkg, nil
}

func (r *salesRepository) ListPackages(ctx context.Context, params pagination.PaginationParams, filters service.SalesPackageListFilters) ([]service.SalesPackage, *pagination.PaginationResult, error) {
	where, args := buildSalesPackageWhere(filters)
	countQuery := `SELECT COUNT(*) FROM sales_packages spkg LEFT JOIN groups g ON g.id = spkg.group_id WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count sales packages: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		salesPackageSelect+` WHERE `+where+`
		 ORDER BY spkg.sort_order ASC, spkg.id DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query sales packages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesPackage, 0)
	for rows.Next() {
		item, scanErr := scanSalesPackage(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan sales package: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sales packages: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *salesRepository) UpsertPartnerPackage(ctx context.Context, mapping *service.SalesPartnerPackage) error {
	return r.runInTx(ctx, func(tx *sql.Tx) error {
		var targetID int64
		if mapping.ID > 0 {
			targetID = mapping.ID
		} else {
			byPairID, err := lookupSalesPartnerPackageID(ctx, tx, mapping.PartnerID, mapping.PackageID, "")
			if err != nil {
				return err
			}
			byCodeID, err := lookupSalesPartnerPackageID(ctx, tx, mapping.PartnerID, 0, mapping.ExternalPackageCode)
			if err != nil {
				return err
			}
			switch {
			case byPairID > 0 && byCodeID > 0 && byPairID != byCodeID:
				return service.ErrSalesPartnerPackageExists
			case byPairID > 0:
				targetID = byPairID
			case byCodeID > 0:
				targetID = byCodeID
			}
		}

		if targetID > 0 {
			result, err := tx.ExecContext(ctx,
				`UPDATE sales_partner_packages
				 SET partner_id = $1, package_id = $2, external_package_code = $3, external_package_name = $4,
				     sale_price = $5, currency = $6, status = $7, updated_at = NOW()
				 WHERE id = $8`,
				mapping.PartnerID, mapping.PackageID, mapping.ExternalPackageCode, mapping.ExternalPackageName,
				mapping.SalePrice, mapping.Currency, mapping.Status, targetID,
			)
			if err != nil {
				if isUniqueViolation(err) {
					return service.ErrSalesPartnerPackageExists.WithCause(err)
				}
				return fmt.Errorf("update sales partner package: %w", err)
			}
			rows, _ := result.RowsAffected()
			if rows == 0 {
				return service.ErrSalesPartnerPackageNotFound
			}
			mapping.ID = targetID
		} else {
			err := tx.QueryRowContext(ctx,
				`INSERT INTO sales_partner_packages
				 (partner_id, package_id, external_package_code, external_package_name, sale_price, currency, status)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)
				 RETURNING id, created_at, updated_at`,
				mapping.PartnerID, mapping.PackageID, mapping.ExternalPackageCode, mapping.ExternalPackageName,
				mapping.SalePrice, mapping.Currency, mapping.Status,
			).Scan(&mapping.ID, &mapping.CreatedAt, &mapping.UpdatedAt)
			if err != nil {
				if isUniqueViolation(err) {
					return service.ErrSalesPartnerPackageExists.WithCause(err)
				}
				return fmt.Errorf("insert sales partner package: %w", err)
			}
			return nil
		}

		loaded, err := scanSalesPartnerPackage(
			tx.QueryRowContext(ctx, salesPartnerPackageSelect+` WHERE spp.id = $1`, mapping.ID),
		)
		if err != nil {
			if err == sql.ErrNoRows {
				return service.ErrSalesPartnerPackageNotFound.WithCause(err)
			}
			return err
		}
		*mapping = *loaded
		return nil
	})
}

func (r *salesRepository) DeletePartnerPackage(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sales_partner_packages WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sales partner package: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesPartnerPackageNotFound
	}
	return nil
}

func (r *salesRepository) GetPartnerPackageByID(ctx context.Context, id int64) (*service.SalesPartnerPackage, error) {
	item, err := scanSalesPartnerPackage(
		r.db.QueryRowContext(ctx, salesPartnerPackageSelect+` WHERE spp.id = $1`, id),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPartnerPackageNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales partner package by id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) GetPartnerPackageByPartnerPackageID(ctx context.Context, partnerID, packageID int64) (*service.SalesPartnerPackage, error) {
	item, err := scanSalesPartnerPackage(
		r.db.QueryRowContext(ctx, salesPartnerPackageSelect+` WHERE spp.partner_id = $1 AND spp.package_id = $2`, partnerID, packageID),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPartnerPackageNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales partner package by partner package id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) GetPartnerPackageByExternalCode(ctx context.Context, partnerID int64, externalCode string) (*service.SalesPartnerPackage, error) {
	item, err := scanSalesPartnerPackage(
		r.db.QueryRowContext(ctx, salesPartnerPackageSelect+` WHERE spp.partner_id = $1 AND spp.external_package_code = $2`, partnerID, externalCode),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesPartnerPackageNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales partner package by external code: %w", err)
	}
	return item, nil
}

func (r *salesRepository) ListPartnerPackages(ctx context.Context, params pagination.PaginationParams, filters service.SalesPartnerPackageListFilters) ([]service.SalesPartnerPackage, *pagination.PaginationResult, error) {
	where, args := buildSalesPartnerPackageWhere(filters)
	countQuery := `SELECT COUNT(*) FROM sales_partner_packages spp
		INNER JOIN sales_partners sp ON sp.id = spp.partner_id
		INNER JOIN sales_packages spkg ON spkg.id = spp.package_id
		LEFT JOIN groups g ON g.id = spkg.group_id
		WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count sales partner packages: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		salesPartnerPackageSelect+` WHERE `+where+`
		 ORDER BY spp.id DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query sales partner packages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesPartnerPackage, 0)
	for rows.Next() {
		item, scanErr := scanSalesPartnerPackage(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan sales partner package: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sales partner packages: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *salesRepository) ListCatalogByPartner(ctx context.Context, partnerID int64, params pagination.PaginationParams, filters service.SalesCatalogFilters) ([]service.SalesCatalogItem, *pagination.PaginationResult, error) {
	where := []string{"spp.partner_id = $1", "spp.status = $2", "spkg.status = $3"}
	args := []any{partnerID, service.StatusActive, service.StatusActive}
	argIdx := 4
	if filters.Platform != "" {
		where = append(where, fmt.Sprintf("spkg.platform = $%d", argIdx))
		args = append(args, filters.Platform)
		argIdx++
	}
	if filters.CycleUnit != "" {
		where = append(where, fmt.Sprintf("spkg.cycle_unit = $%d", argIdx))
		args = append(args, filters.CycleUnit)
		argIdx++
	}
	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sales_partner_packages spp
		 INNER JOIN sales_packages spkg ON spkg.id = spp.package_id
		 WHERE `+whereClause,
		args...,
	).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count sales partner catalog: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		`SELECT
			spp.id, spp.partner_id, spp.package_id, spp.external_package_code, spp.external_package_name,
			spp.sale_price, spp.currency, spp.status,
			spkg.id, spkg.code, spkg.name, spkg.description, spkg.platform, spkg.group_id,
			spkg.cycle_unit, spkg.cycle_count, spkg.validity_days, spkg.key_policy,
			spkg.auto_create_key, spkg.status, spkg.store_visible, spkg.sort_order,
			spkg.created_at, spkg.updated_at,
			g.id, g.name, g.description, g.platform, g.status, g.subscription_type, g.default_validity_days
		 FROM sales_partner_packages spp
		 INNER JOIN sales_packages spkg ON spkg.id = spp.package_id
		 LEFT JOIN groups g ON g.id = spkg.group_id
		 WHERE `+whereClause+`
		 ORDER BY spkg.sort_order ASC, spkg.id ASC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query sales partner catalog: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesCatalogItem, 0)
	for rows.Next() {
		item, scanErr := scanSalesCatalogItem(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan sales catalog item: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sales partner catalog: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *salesRepository) ListStoreCatalog(ctx context.Context, params pagination.PaginationParams, filters service.SalesCatalogFilters) ([]service.SalesPackage, *pagination.PaginationResult, error) {
	where := []string{"spkg.status = $1", "spkg.store_visible = TRUE"}
	args := []any{service.StatusActive}
	argIdx := 2
	if filters.Platform != "" {
		where = append(where, fmt.Sprintf("spkg.platform = $%d", argIdx))
		args = append(args, filters.Platform)
		argIdx++
	}
	if filters.CycleUnit != "" {
		where = append(where, fmt.Sprintf("spkg.cycle_unit = $%d", argIdx))
		args = append(args, filters.CycleUnit)
		argIdx++
	}
	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sales_packages spkg WHERE `+whereClause,
		args...,
	).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count store sales catalog: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		salesPackageSelect+` WHERE `+whereClause+`
		 ORDER BY spkg.sort_order ASC, spkg.id ASC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query store sales catalog: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesPackage, 0)
	for rows.Next() {
		item, scanErr := scanSalesPackage(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan store sales catalog: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate store sales catalog: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *salesRepository) GetBindingByID(ctx context.Context, id int64) (*service.SalesUserBinding, error) {
	item, err := scanSalesBindingWithRelations(
		r.db.QueryRowContext(ctx, salesBindingSelect+` WHERE sb.id = $1`, id),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesBindingNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales binding by id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) GetBindingByPartnerExternalUserID(ctx context.Context, partnerID int64, externalUserID string) (*service.SalesUserBinding, error) {
	item, err := scanSalesBinding(
		r.db.QueryRowContext(ctx,
			`SELECT id, partner_id, external_user_id, user_id, external_email, external_name, metadata, created_at, updated_at
			 FROM sales_user_bindings
			 WHERE partner_id = $1 AND external_user_id = $2
			 LIMIT 1`,
			partnerID, externalUserID,
		),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesBindingNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales binding by external user id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) GetBindingByPartnerUserID(ctx context.Context, partnerID, userID int64) (*service.SalesUserBinding, error) {
	item, err := scanSalesBinding(
		r.db.QueryRowContext(ctx,
			`SELECT id, partner_id, external_user_id, user_id, external_email, external_name, metadata, created_at, updated_at
			 FROM sales_user_bindings
			 WHERE partner_id = $1 AND user_id = $2
			 ORDER BY updated_at DESC, id DESC
			 LIMIT 1`,
			partnerID, userID,
		),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesBindingNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales binding by user id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) ListBindings(ctx context.Context, params pagination.PaginationParams, filters service.SalesBindingListFilters) ([]service.SalesUserBinding, *pagination.PaginationResult, error) {
	where, args := buildSalesBindingWhere(filters)
	countQuery := `SELECT COUNT(*) FROM sales_user_bindings sb
		INNER JOIN sales_partners sp ON sp.id = sb.partner_id
		INNER JOIN users u ON u.id = sb.user_id
		WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count sales bindings: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		salesBindingSelect+` WHERE `+where+`
		 ORDER BY sb.updated_at DESC, sb.id DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query sales bindings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesUserBinding, 0)
	for rows.Next() {
		item, scanErr := scanSalesBindingWithRelations(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan sales binding: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sales bindings: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

func (r *salesRepository) CreateBinding(ctx context.Context, binding *service.SalesUserBinding) error {
	metadata, err := marshalSalesMap(binding.Metadata)
	if err != nil {
		return fmt.Errorf("marshal sales binding metadata: %w", err)
	}
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO sales_user_bindings
		 (partner_id, external_user_id, user_id, external_email, external_name, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		binding.PartnerID, binding.ExternalUserID, binding.UserID, binding.ExternalEmail, binding.ExternalName, metadata,
	).Scan(&binding.ID, &binding.CreatedAt, &binding.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesBindingConflict.WithCause(err)
		}
		return fmt.Errorf("insert sales binding: %w", err)
	}
	return nil
}

func (r *salesRepository) UpdateBinding(ctx context.Context, binding *service.SalesUserBinding) error {
	metadata, err := marshalSalesMap(binding.Metadata)
	if err != nil {
		return fmt.Errorf("marshal sales binding metadata: %w", err)
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE sales_user_bindings
		 SET partner_id = $1, external_user_id = $2, user_id = $3, external_email = $4,
		     external_name = $5, metadata = $6, updated_at = NOW()
		 WHERE id = $7`,
		binding.PartnerID, binding.ExternalUserID, binding.UserID, binding.ExternalEmail, binding.ExternalName, metadata, binding.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesBindingConflict.WithCause(err)
		}
		return fmt.Errorf("update sales binding: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesBindingNotFound
	}
	current, err := r.GetBindingByID(ctx, binding.ID)
	if err != nil {
		return err
	}
	*binding = *current
	return nil
}

func (r *salesRepository) DeleteBinding(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sales_user_bindings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sales binding: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesBindingNotFound
	}
	return nil
}

func (r *salesRepository) CreateOrder(ctx context.Context, order *service.SalesOrder) error {
	packageSnapshot, err := marshalSalesMap(order.PackageSnapshot)
	if err != nil {
		return fmt.Errorf("marshal sales order package snapshot: %w", err)
	}
	rawPayload, err := marshalSalesMap(order.RawPayload)
	if err != nil {
		return fmt.Errorf("marshal sales order raw payload: %w", err)
	}
	resultSnapshot, err := marshalSalesMap(order.ResultSnapshot)
	if err != nil {
		return fmt.Errorf("marshal sales order result snapshot: %w", err)
	}
	err = r.db.QueryRowContext(ctx,
		`INSERT INTO sales_orders
		 (partner_id, external_order_id, external_user_id, user_id, package_id, order_type, status,
		  subscription_id, api_key_id, amount, currency, package_snapshot, raw_payload, result_snapshot,
		  error_message, fulfilled_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7,
		         $8, $9, $10, $11, $12, $13, $14,
		         $15, $16)
		 RETURNING id, created_at, updated_at`,
		order.PartnerID, order.ExternalOrderID, order.ExternalUserID, nullableInt64Ptr(order.UserID), order.PackageID,
		order.OrderType, order.Status, nullableInt64Ptr(order.SubscriptionID), nullableInt64Ptr(order.APIKeyID),
		order.Amount, order.Currency, packageSnapshot, rawPayload, resultSnapshot, order.ErrorMessage, order.FulfilledAt,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesOrderExists.WithCause(err)
		}
		return fmt.Errorf("insert sales order: %w", err)
	}
	return nil
}

func (r *salesRepository) UpdateOrder(ctx context.Context, order *service.SalesOrder) error {
	packageSnapshot, err := marshalSalesMap(order.PackageSnapshot)
	if err != nil {
		return fmt.Errorf("marshal sales order package snapshot: %w", err)
	}
	rawPayload, err := marshalSalesMap(order.RawPayload)
	if err != nil {
		return fmt.Errorf("marshal sales order raw payload: %w", err)
	}
	resultSnapshot, err := marshalSalesMap(order.ResultSnapshot)
	if err != nil {
		return fmt.Errorf("marshal sales order result snapshot: %w", err)
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE sales_orders
		 SET partner_id = $1, external_order_id = $2, external_user_id = $3, user_id = $4, package_id = $5,
		     order_type = $6, status = $7, subscription_id = $8, api_key_id = $9, amount = $10, currency = $11,
		     package_snapshot = $12, raw_payload = $13, result_snapshot = $14, error_message = $15,
		     fulfilled_at = $16, updated_at = NOW()
		 WHERE id = $17`,
		order.PartnerID, order.ExternalOrderID, order.ExternalUserID, nullableInt64Ptr(order.UserID), order.PackageID,
		order.OrderType, order.Status, nullableInt64Ptr(order.SubscriptionID), nullableInt64Ptr(order.APIKeyID),
		order.Amount, order.Currency, packageSnapshot, rawPayload, resultSnapshot, order.ErrorMessage, order.FulfilledAt, order.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrSalesOrderExists.WithCause(err)
		}
		return fmt.Errorf("update sales order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesOrderNotFound
	}
	current, err := r.GetOrderByID(ctx, order.ID)
	if err != nil {
		return err
	}
	*order = *current
	return nil
}

func (r *salesRepository) DeleteOrder(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM sales_orders WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete sales order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesOrderNotFound
	}
	return nil
}

func (r *salesRepository) DeleteOrders(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	params := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		params = append(params, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}
	query := fmt.Sprintf(`DELETE FROM sales_orders WHERE id IN (%s)`, strings.Join(params, ","))
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch delete sales orders: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return service.ErrSalesOrderNotFound
	}
	return nil
}

func (r *salesRepository) GetOrderByID(ctx context.Context, id int64) (*service.SalesOrder, error) {
	item, err := scanSalesOrder(
		r.db.QueryRowContext(ctx, salesOrderSelect+` WHERE so.id = $1`, id),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesOrderNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales order by id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) GetOrderByPartnerExternalID(ctx context.Context, partnerID int64, externalOrderID string) (*service.SalesOrder, error) {
	item, err := scanSalesOrder(
		r.db.QueryRowContext(ctx, salesOrderSelect+` WHERE so.partner_id = $1 AND so.external_order_id = $2`, partnerID, externalOrderID),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesOrderNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get sales order by partner external id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) GetLatestFulfilledOrderByPartnerAPIKeyID(ctx context.Context, partnerID, apiKeyID int64) (*service.SalesOrder, error) {
	item, err := scanSalesOrder(
		r.db.QueryRowContext(
			ctx,
			salesOrderSelect+` WHERE so.partner_id = $1 AND so.api_key_id = $2 AND so.status = $3
			 ORDER BY so.created_at DESC, so.id DESC
			 LIMIT 1`,
			partnerID,
			apiKeyID,
			service.SalesOrderStatusFulfilled,
		),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrSalesOrderNotFound.WithCause(err)
		}
		return nil, fmt.Errorf("get latest fulfilled sales order by partner api key id: %w", err)
	}
	return item, nil
}

func (r *salesRepository) ListOrders(ctx context.Context, params pagination.PaginationParams, filters service.SalesOrderListFilters) ([]service.SalesOrder, *pagination.PaginationResult, error) {
	where, args := buildSalesOrderWhere(filters)
	countQuery := `SELECT COUNT(*) FROM sales_orders so
		INNER JOIN sales_partners sp ON sp.id = so.partner_id
		INNER JOIN sales_packages spkg ON spkg.id = so.package_id
		LEFT JOIN groups g ON g.id = spkg.group_id
		LEFT JOIN users u ON u.id = so.user_id
		WHERE ` + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count sales orders: %w", err)
	}

	args = append(args, params.Limit(), params.Offset())
	rows, err := r.db.QueryContext(ctx,
		salesOrderSelect+` WHERE `+where+`
		 ORDER BY so.created_at DESC, so.id DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query sales orders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.SalesOrder, 0)
	for rows.Next() {
		item, scanErr := scanSalesOrder(rows)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("scan sales order: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate sales orders: %w", err)
	}
	return items, paginationResultFromTotal(total, params), nil
}

const salesPackageSelect = `
SELECT
	spkg.id, spkg.code, spkg.name, spkg.description, spkg.platform, spkg.group_id,
	spkg.cycle_unit, spkg.cycle_count, spkg.validity_days, spkg.key_policy,
	spkg.auto_create_key, spkg.status, spkg.store_visible, spkg.sort_order,
	spkg.created_at, spkg.updated_at,
	g.id, g.name, g.description, g.platform, g.status, g.subscription_type, g.default_validity_days
FROM sales_packages spkg
LEFT JOIN groups g ON g.id = spkg.group_id
`

const salesPartnerPackageSelect = `
SELECT
	spp.id, spp.partner_id, spp.package_id, spp.external_package_code, spp.external_package_name,
	spp.sale_price, spp.currency, spp.status, spp.created_at, spp.updated_at,
	sp.id, sp.code, sp.name, sp.description, sp.status, sp.auth_mode, sp.secret_hash, sp.secret_hint, sp.rate_limit_rpm, sp.created_at, sp.updated_at,
	spkg.id, spkg.code, spkg.name, spkg.description, spkg.platform, spkg.group_id,
	spkg.cycle_unit, spkg.cycle_count, spkg.validity_days, spkg.key_policy,
	spkg.auto_create_key, spkg.status, spkg.store_visible, spkg.sort_order,
	spkg.created_at, spkg.updated_at,
	g.id, g.name, g.description, g.platform, g.status, g.subscription_type, g.default_validity_days
FROM sales_partner_packages spp
INNER JOIN sales_partners sp ON sp.id = spp.partner_id
INNER JOIN sales_packages spkg ON spkg.id = spp.package_id
LEFT JOIN groups g ON g.id = spkg.group_id
`

const salesOrderSelect = `
SELECT
	so.id, so.partner_id, so.external_order_id, so.external_user_id, so.user_id, so.package_id,
	so.order_type, so.status, so.subscription_id, so.api_key_id, COALESCE(spp.sale_price, so.amount), so.currency,
	so.package_snapshot, so.raw_payload, so.result_snapshot, so.error_message, so.fulfilled_at,
	so.created_at, so.updated_at,
	sp.id, sp.code, sp.name, sp.description, sp.status, sp.auth_mode, sp.secret_hash, sp.secret_hint, sp.rate_limit_rpm, sp.created_at, sp.updated_at,
	spkg.id, spkg.code, spkg.name, spkg.description, spkg.platform, spkg.group_id,
	spkg.cycle_unit, spkg.cycle_count, spkg.validity_days, spkg.key_policy,
	spkg.auto_create_key, spkg.status, spkg.store_visible, spkg.sort_order,
	spkg.created_at, spkg.updated_at,
	g.id, g.name, g.description, g.platform, g.status, g.subscription_type, g.default_validity_days,
	u.id, u.email, u.username, u.role, u.status
FROM sales_orders so
INNER JOIN sales_partners sp ON sp.id = so.partner_id
INNER JOIN sales_packages spkg ON spkg.id = so.package_id
LEFT JOIN sales_partner_packages spp ON spp.partner_id = so.partner_id AND spp.package_id = so.package_id
LEFT JOIN groups g ON g.id = spkg.group_id
LEFT JOIN users u ON u.id = so.user_id
`

const salesBindingSelect = `
SELECT
	sb.id, sb.partner_id, sb.external_user_id, sb.user_id, sb.external_email, sb.external_name, sb.metadata, sb.created_at, sb.updated_at,
	sp.id, sp.code, sp.name, sp.description, sp.status, sp.auth_mode, sp.secret_hash, sp.secret_hint, sp.rate_limit_rpm, sp.created_at, sp.updated_at,
	u.id, u.email, u.username, u.role, u.status
FROM sales_user_bindings sb
INNER JOIN sales_partners sp ON sp.id = sb.partner_id
INNER JOIN users u ON u.id = sb.user_id
`

func scanSalesPartner(scanner rowScanner) (*service.SalesPartner, error) {
	item := &service.SalesPartner{}
	if err := scanner.Scan(
		&item.ID, &item.Code, &item.Name, &item.Description, &item.Status, &item.AuthMode,
		&item.SecretHash, &item.SecretHint, &item.RateLimitRPM, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}

func scanSalesPackage(scanner rowScanner) (*service.SalesPackage, error) {
	item := &service.SalesPackage{}
	var (
		groupID           sql.NullInt64
		groupName         sql.NullString
		groupDescription  sql.NullString
		groupPlatform     sql.NullString
		groupStatus       sql.NullString
		groupSubscription sql.NullString
		groupValidityDays sql.NullInt64
	)
	if err := scanner.Scan(
		&item.ID, &item.Code, &item.Name, &item.Description, &item.Platform, &item.GroupID,
		&item.CycleUnit, &item.CycleCount, &item.ValidityDays, &item.KeyPolicy,
		&item.AutoCreateKey, &item.Status, &item.StoreVisible, &item.SortOrder,
		&item.CreatedAt, &item.UpdatedAt,
		&groupID, &groupName, &groupDescription, &groupPlatform, &groupStatus, &groupSubscription, &groupValidityDays,
	); err != nil {
		return nil, err
	}
	if groupID.Valid {
		item.Group = &service.Group{
			ID:                  groupID.Int64,
			Name:                groupName.String,
			Description:         groupDescription.String,
			Platform:            groupPlatform.String,
			Status:              groupStatus.String,
			SubscriptionType:    groupSubscription.String,
			DefaultValidityDays: int(groupValidityDays.Int64),
			Hydrated:            true,
		}
	}
	return item, nil
}

func scanSalesPartnerPackage(scanner rowScanner) (*service.SalesPartnerPackage, error) {
	item := &service.SalesPartnerPackage{}
	item.Partner = &service.SalesPartner{}
	pkg := &service.SalesPackage{}
	var (
		groupID           sql.NullInt64
		groupName         sql.NullString
		groupDescription  sql.NullString
		groupPlatform     sql.NullString
		groupStatus       sql.NullString
		groupSubscription sql.NullString
		groupValidityDays sql.NullInt64
	)
	if err := scanner.Scan(
		&item.ID, &item.PartnerID, &item.PackageID, &item.ExternalPackageCode, &item.ExternalPackageName,
		&item.SalePrice, &item.Currency, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		&item.Partner.ID, &item.Partner.Code, &item.Partner.Name, &item.Partner.Description, &item.Partner.Status,
		&item.Partner.AuthMode, &item.Partner.SecretHash, &item.Partner.SecretHint, &item.Partner.RateLimitRPM,
		&item.Partner.CreatedAt, &item.Partner.UpdatedAt,
		&pkg.ID, &pkg.Code, &pkg.Name, &pkg.Description, &pkg.Platform, &pkg.GroupID,
		&pkg.CycleUnit, &pkg.CycleCount, &pkg.ValidityDays, &pkg.KeyPolicy,
		&pkg.AutoCreateKey, &pkg.Status, &pkg.StoreVisible, &pkg.SortOrder,
		&pkg.CreatedAt, &pkg.UpdatedAt,
		&groupID, &groupName, &groupDescription, &groupPlatform, &groupStatus, &groupSubscription, &groupValidityDays,
	); err != nil {
		return nil, err
	}
	if groupID.Valid {
		pkg.Group = &service.Group{
			ID:                  groupID.Int64,
			Name:                groupName.String,
			Description:         groupDescription.String,
			Platform:            groupPlatform.String,
			Status:              groupStatus.String,
			SubscriptionType:    groupSubscription.String,
			DefaultValidityDays: int(groupValidityDays.Int64),
			Hydrated:            true,
		}
	}
	item.Package = pkg
	return item, nil
}

func scanSalesCatalogItem(scanner rowScanner) (*service.SalesCatalogItem, error) {
	item := &service.SalesCatalogItem{}
	pkg := &service.SalesPackage{}
	var (
		groupID           sql.NullInt64
		groupName         sql.NullString
		groupDescription  sql.NullString
		groupPlatform     sql.NullString
		groupStatus       sql.NullString
		groupSubscription sql.NullString
		groupValidityDays sql.NullInt64
	)
	if err := scanner.Scan(
		&item.MappingID, &item.PartnerID, &item.PackageID, &item.ExternalPackageCode, &item.ExternalPackageName,
		&item.SalePrice, &item.Currency, &item.PartnerPackageStatus,
		&pkg.ID, &pkg.Code, &pkg.Name, &pkg.Description, &pkg.Platform, &pkg.GroupID,
		&pkg.CycleUnit, &pkg.CycleCount, &pkg.ValidityDays, &pkg.KeyPolicy,
		&pkg.AutoCreateKey, &pkg.Status, &pkg.StoreVisible, &pkg.SortOrder,
		&pkg.CreatedAt, &pkg.UpdatedAt,
		&groupID, &groupName, &groupDescription, &groupPlatform, &groupStatus, &groupSubscription, &groupValidityDays,
	); err != nil {
		return nil, err
	}
	if groupID.Valid {
		pkg.Group = &service.Group{
			ID:                  groupID.Int64,
			Name:                groupName.String,
			Description:         groupDescription.String,
			Platform:            groupPlatform.String,
			Status:              groupStatus.String,
			SubscriptionType:    groupSubscription.String,
			DefaultValidityDays: int(groupValidityDays.Int64),
			Hydrated:            true,
		}
	}
	item.Package = pkg
	return item, nil
}

func scanSalesBinding(scanner rowScanner) (*service.SalesUserBinding, error) {
	item := &service.SalesUserBinding{}
	var metadata []byte
	if err := scanner.Scan(
		&item.ID, &item.PartnerID, &item.ExternalUserID, &item.UserID, &item.ExternalEmail, &item.ExternalName,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = unmarshalSalesMap(metadata)
	return item, nil
}

func scanSalesBindingWithRelations(scanner rowScanner) (*service.SalesUserBinding, error) {
	item := &service.SalesUserBinding{}
	item.Partner = &service.SalesPartner{}
	item.User = &service.User{}
	var metadata []byte
	if err := scanner.Scan(
		&item.ID, &item.PartnerID, &item.ExternalUserID, &item.UserID, &item.ExternalEmail, &item.ExternalName,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
		&item.Partner.ID, &item.Partner.Code, &item.Partner.Name, &item.Partner.Description, &item.Partner.Status,
		&item.Partner.AuthMode, &item.Partner.SecretHash, &item.Partner.SecretHint, &item.Partner.RateLimitRPM,
		&item.Partner.CreatedAt, &item.Partner.UpdatedAt,
		&item.User.ID, &item.User.Email, &item.User.Username, &item.User.Role, &item.User.Status,
	); err != nil {
		return nil, err
	}
	item.Metadata = unmarshalSalesMap(metadata)
	return item, nil
}

func scanSalesOrder(scanner rowScanner) (*service.SalesOrder, error) {
	item := &service.SalesOrder{}
	item.Partner = &service.SalesPartner{}
	pkg := &service.SalesPackage{}
	var (
		userID            sql.NullInt64
		subscriptionID    sql.NullInt64
		apiKeyID          sql.NullInt64
		fulfilledAt       sql.NullTime
		packageSnapshot   []byte
		rawPayload        []byte
		resultSnapshot    []byte
		groupID           sql.NullInt64
		groupName         sql.NullString
		groupDescription  sql.NullString
		groupPlatform     sql.NullString
		groupStatus       sql.NullString
		groupSubscription sql.NullString
		groupValidityDays sql.NullInt64
		orderUserID       sql.NullInt64
		orderUserEmail    sql.NullString
		orderUserUsername sql.NullString
		orderUserRole     sql.NullString
		orderUserStatus   sql.NullString
	)
	if err := scanner.Scan(
		&item.ID, &item.PartnerID, &item.ExternalOrderID, &item.ExternalUserID, &userID, &item.PackageID,
		&item.OrderType, &item.Status, &subscriptionID, &apiKeyID, &item.Amount, &item.Currency,
		&packageSnapshot, &rawPayload, &resultSnapshot, &item.ErrorMessage, &fulfilledAt,
		&item.CreatedAt, &item.UpdatedAt,
		&item.Partner.ID, &item.Partner.Code, &item.Partner.Name, &item.Partner.Description, &item.Partner.Status,
		&item.Partner.AuthMode, &item.Partner.SecretHash, &item.Partner.SecretHint, &item.Partner.RateLimitRPM,
		&item.Partner.CreatedAt, &item.Partner.UpdatedAt,
		&pkg.ID, &pkg.Code, &pkg.Name, &pkg.Description, &pkg.Platform, &pkg.GroupID,
		&pkg.CycleUnit, &pkg.CycleCount, &pkg.ValidityDays, &pkg.KeyPolicy,
		&pkg.AutoCreateKey, &pkg.Status, &pkg.StoreVisible, &pkg.SortOrder,
		&pkg.CreatedAt, &pkg.UpdatedAt,
		&groupID, &groupName, &groupDescription, &groupPlatform, &groupStatus, &groupSubscription, &groupValidityDays,
		&orderUserID, &orderUserEmail, &orderUserUsername, &orderUserRole, &orderUserStatus,
	); err != nil {
		return nil, err
	}
	if userID.Valid {
		v := userID.Int64
		item.UserID = &v
	}
	if subscriptionID.Valid {
		v := subscriptionID.Int64
		item.SubscriptionID = &v
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		item.APIKeyID = &v
	}
	if fulfilledAt.Valid {
		v := fulfilledAt.Time
		item.FulfilledAt = &v
	}
	item.PackageSnapshot = unmarshalSalesMap(packageSnapshot)
	item.RawPayload = unmarshalSalesMap(rawPayload)
	item.ResultSnapshot = unmarshalSalesMap(resultSnapshot)
	if groupID.Valid {
		pkg.Group = &service.Group{
			ID:                  groupID.Int64,
			Name:                groupName.String,
			Description:         groupDescription.String,
			Platform:            groupPlatform.String,
			Status:              groupStatus.String,
			SubscriptionType:    groupSubscription.String,
			DefaultValidityDays: int(groupValidityDays.Int64),
			Hydrated:            true,
		}
	}
	item.Package = pkg
	if orderUserID.Valid {
		item.User = &service.User{
			ID:       orderUserID.Int64,
			Email:    orderUserEmail.String,
			Username: orderUserUsername.String,
			Role:     orderUserRole.String,
			Status:   orderUserStatus.String,
		}
	}
	return item, nil
}

func marshalSalesMap(in map[string]any) ([]byte, error) {
	if len(in) == 0 {
		return []byte(`{}`), nil
	}
	return json.Marshal(in)
}

func unmarshalSalesMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}

func nullableInt64Ptr(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func lookupSalesPartnerPackageID(ctx context.Context, tx *sql.Tx, partnerID, packageID int64, externalCode string) (int64, error) {
	var (
		query string
		args  []any
		id    int64
	)
	switch {
	case packageID > 0:
		query = `SELECT id FROM sales_partner_packages WHERE partner_id = $1 AND package_id = $2`
		args = []any{partnerID, packageID}
	case externalCode != "":
		query = `SELECT id FROM sales_partner_packages WHERE partner_id = $1 AND external_package_code = $2`
		args = []any{partnerID, externalCode}
	default:
		return 0, nil
	}
	err := tx.QueryRowContext(ctx, query, args...).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("lookup sales partner package: %w", err)
	}
	return id, nil
}

func buildSalesPartnerWhere(filters service.SalesPartnerListFilters) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0)
	argIdx := 1
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("sp.status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		where = append(where, fmt.Sprintf("(sp.code ILIKE $%d OR sp.name ILIKE $%d OR sp.description ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, like)
	}
	return strings.Join(where, " AND "), args
}

func buildSalesPackageWhere(filters service.SalesPackageListFilters) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0)
	argIdx := 1
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("spkg.status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Platform != "" {
		where = append(where, fmt.Sprintf("spkg.platform = $%d", argIdx))
		args = append(args, filters.Platform)
		argIdx++
	}
	if filters.StoreVisible != nil {
		where = append(where, fmt.Sprintf("spkg.store_visible = $%d", argIdx))
		args = append(args, *filters.StoreVisible)
		argIdx++
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		where = append(where, fmt.Sprintf("(spkg.code ILIKE $%d OR spkg.name ILIKE $%d OR spkg.description ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, like)
	}
	return strings.Join(where, " AND "), args
}

func buildSalesBindingWhere(filters service.SalesBindingListFilters) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0)
	argIdx := 1
	if filters.PartnerID != nil && *filters.PartnerID > 0 {
		where = append(where, fmt.Sprintf("sb.partner_id = $%d", argIdx))
		args = append(args, *filters.PartnerID)
		argIdx++
	}
	if filters.UserID != nil && *filters.UserID > 0 {
		where = append(where, fmt.Sprintf("sb.user_id = $%d", argIdx))
		args = append(args, *filters.UserID)
		argIdx++
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		where = append(where, fmt.Sprintf(`(
			sb.external_user_id ILIKE $%d OR
			sb.external_email ILIKE $%d OR
			sb.external_name ILIKE $%d OR
			sp.code ILIKE $%d OR
			sp.name ILIKE $%d OR
			u.email ILIKE $%d OR
			u.username ILIKE $%d
		)`, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, like)
	}
	return strings.Join(where, " AND "), args
}

func buildSalesPartnerPackageWhere(filters service.SalesPartnerPackageListFilters) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0)
	argIdx := 1
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("spp.status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.PartnerID != nil {
		where = append(where, fmt.Sprintf("spp.partner_id = $%d", argIdx))
		args = append(args, *filters.PartnerID)
		argIdx++
	}
	if filters.PackageID != nil {
		where = append(where, fmt.Sprintf("spp.package_id = $%d", argIdx))
		args = append(args, *filters.PackageID)
	}
	return strings.Join(where, " AND "), args
}

func buildSalesOrderWhere(filters service.SalesOrderListFilters) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0)
	argIdx := 1
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("so.status = $%d", argIdx))
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.PartnerID != nil {
		where = append(where, fmt.Sprintf("so.partner_id = $%d", argIdx))
		args = append(args, *filters.PartnerID)
		argIdx++
	}
	if filters.PackageID != nil {
		where = append(where, fmt.Sprintf("so.package_id = $%d", argIdx))
		args = append(args, *filters.PackageID)
		argIdx++
	}
	if filters.UserID != nil {
		where = append(where, fmt.Sprintf("so.user_id = $%d", argIdx))
		args = append(args, *filters.UserID)
		argIdx++
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		like := "%" + escapeLike(search) + "%"
		where = append(where, fmt.Sprintf(`(
			so.external_order_id ILIKE $%d OR
			so.external_user_id ILIKE $%d OR
			sp.code ILIKE $%d OR
			sp.name ILIKE $%d OR
			spkg.code ILIKE $%d OR
			spkg.name ILIKE $%d OR
			COALESCE(u.email, '') ILIKE $%d OR
			COALESCE(u.username, '') ILIKE $%d
		)`, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, like)
	}
	return strings.Join(where, " AND "), args
}
