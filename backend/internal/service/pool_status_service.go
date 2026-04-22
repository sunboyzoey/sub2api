package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PoolStatusOverview represents the overall health summary of the account pool.
type PoolStatusOverview struct {
	TotalAccounts            int64   `json:"total_accounts"`
	AvailableAccounts        int64   `json:"available_accounts"`
	RateLimitedAccounts      int64   `json:"rate_limited_accounts"`
	OverloadedAccounts       int64   `json:"overloaded_accounts"`
	TempUnschedulableAccount int64   `json:"temp_unschedulable_accounts"`
	PausedAccounts           int64   `json:"paused_accounts"`
	ErrorAccounts            int64   `json:"error_accounts"`
	DisabledAccounts         int64   `json:"disabled_accounts"`
	AvailabilityRatio        float64 `json:"availability_ratio"`
}

// PoolStatusPlatform represents aggregated account health per platform.
type PoolStatusPlatform struct {
	Platform                 string  `json:"platform"`
	TotalAccounts            int64   `json:"total_accounts"`
	AvailableAccounts        int64   `json:"available_accounts"`
	RateLimitedAccounts      int64   `json:"rate_limited_accounts"`
	OverloadedAccounts       int64   `json:"overloaded_accounts"`
	TempUnschedulableAccount int64   `json:"temp_unschedulable_accounts"`
	PausedAccounts           int64   `json:"paused_accounts"`
	ErrorAccounts            int64   `json:"error_accounts"`
	DisabledAccounts         int64   `json:"disabled_accounts"`
	AvailabilityRatio        float64 `json:"availability_ratio"`
}

// PoolStatusGroup represents aggregated account health per group.
type PoolStatusGroup struct {
	GroupID                  int64   `json:"group_id"`
	GroupName                string  `json:"group_name"`
	Platform                 string  `json:"platform"`
	Description              string  `json:"description"`
	TotalAccounts            int64   `json:"total_accounts"`
	AvailableAccounts        int64   `json:"available_accounts"`
	RateLimitedAccounts      int64   `json:"rate_limited_accounts"`
	OverloadedAccounts       int64   `json:"overloaded_accounts"`
	TempUnschedulableAccount int64   `json:"temp_unschedulable_accounts"`
	PausedAccounts           int64   `json:"paused_accounts"`
	ErrorAccounts            int64   `json:"error_accounts"`
	DisabledAccounts         int64   `json:"disabled_accounts"`
	AvailabilityRatio        float64 `json:"availability_ratio"`
}

// PoolStatusSummary is the public response payload for the pool status page.
type PoolStatusSummary struct {
	GeneratedAt time.Time            `json:"generated_at"`
	Overview    PoolStatusOverview   `json:"overview"`
	Platforms   []PoolStatusPlatform `json:"platforms"`
	Groups      []PoolStatusGroup    `json:"groups"`
}

// PoolStatusService provides a lightweight public account-pool health summary.
type PoolStatusService struct {
	db *sql.DB
}

// NewPoolStatusService creates a new PoolStatusService.
func NewPoolStatusService(db *sql.DB) *PoolStatusService {
	return &PoolStatusService{db: db}
}

func (s *PoolStatusService) GetSummary(ctx context.Context) (*PoolStatusSummary, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("pool status service is not initialized")
	}

	overview, err := s.queryOverview(ctx)
	if err != nil {
		return nil, err
	}

	platforms, err := s.queryPlatforms(ctx)
	if err != nil {
		return nil, err
	}

	groups, err := s.queryGroups(ctx)
	if err != nil {
		return nil, err
	}

	return &PoolStatusSummary{
		GeneratedAt: time.Now(),
		Overview:    overview,
		Platforms:   platforms,
		Groups:      groups,
	}, nil
}

func (s *PoolStatusService) queryOverview(ctx context.Context) (PoolStatusOverview, error) {
	const query = `
WITH classified AS (
	SELECT
		CASE
			WHEN status = 'error' THEN 'error'
			WHEN status <> 'active' THEN 'disabled'
			WHEN rate_limit_reset_at IS NOT NULL AND rate_limit_reset_at > NOW() THEN 'rate_limited'
			WHEN overload_until IS NOT NULL AND overload_until > NOW() THEN 'overloaded'
			WHEN temp_unschedulable_until IS NOT NULL AND temp_unschedulable_until > NOW() THEN 'temp_unschedulable'
			WHEN schedulable = TRUE AND (
				auto_pause_on_expired = FALSE OR
				expires_at IS NULL OR
				expires_at > NOW()
			) THEN 'available'
			ELSE 'paused'
		END AS runtime_status
	FROM accounts
	WHERE deleted_at IS NULL
)
SELECT
	COUNT(*) AS total_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'available') AS available_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'rate_limited') AS rate_limited_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'overloaded') AS overloaded_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'temp_unschedulable') AS temp_unschedulable_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'paused') AS paused_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'error') AS error_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'disabled') AS disabled_accounts
FROM classified
`

	var out PoolStatusOverview
	err := s.db.QueryRowContext(ctx, query).Scan(
		&out.TotalAccounts,
		&out.AvailableAccounts,
		&out.RateLimitedAccounts,
		&out.OverloadedAccounts,
		&out.TempUnschedulableAccount,
		&out.PausedAccounts,
		&out.ErrorAccounts,
		&out.DisabledAccounts,
	)
	if err != nil {
		return PoolStatusOverview{}, fmt.Errorf("query pool overview: %w", err)
	}
	out.AvailabilityRatio = calculateAvailabilityRatio(out.AvailableAccounts, out.TotalAccounts)
	return out, nil
}

func (s *PoolStatusService) queryPlatforms(ctx context.Context) ([]PoolStatusPlatform, error) {
	const query = `
WITH classified AS (
	SELECT
		platform,
		CASE
			WHEN status = 'error' THEN 'error'
			WHEN status <> 'active' THEN 'disabled'
			WHEN rate_limit_reset_at IS NOT NULL AND rate_limit_reset_at > NOW() THEN 'rate_limited'
			WHEN overload_until IS NOT NULL AND overload_until > NOW() THEN 'overloaded'
			WHEN temp_unschedulable_until IS NOT NULL AND temp_unschedulable_until > NOW() THEN 'temp_unschedulable'
			WHEN schedulable = TRUE AND (
				auto_pause_on_expired = FALSE OR
				expires_at IS NULL OR
				expires_at > NOW()
			) THEN 'available'
			ELSE 'paused'
		END AS runtime_status
	FROM accounts
	WHERE deleted_at IS NULL
)
SELECT
	platform,
	COUNT(*) AS total_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'available') AS available_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'rate_limited') AS rate_limited_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'overloaded') AS overloaded_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'temp_unschedulable') AS temp_unschedulable_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'paused') AS paused_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'error') AS error_accounts,
	COUNT(*) FILTER (WHERE runtime_status = 'disabled') AS disabled_accounts
FROM classified
GROUP BY platform
ORDER BY available_accounts DESC, total_accounts DESC, platform ASC
`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query pool platforms: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]PoolStatusPlatform, 0)
	for rows.Next() {
		var item PoolStatusPlatform
		if err := rows.Scan(
			&item.Platform,
			&item.TotalAccounts,
			&item.AvailableAccounts,
			&item.RateLimitedAccounts,
			&item.OverloadedAccounts,
			&item.TempUnschedulableAccount,
			&item.PausedAccounts,
			&item.ErrorAccounts,
			&item.DisabledAccounts,
		); err != nil {
			return nil, fmt.Errorf("scan pool platform: %w", err)
		}
		item.AvailabilityRatio = calculateAvailabilityRatio(item.AvailableAccounts, item.TotalAccounts)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pool platforms: %w", err)
	}

	return items, nil
}

func (s *PoolStatusService) queryGroups(ctx context.Context) ([]PoolStatusGroup, error) {
	const query = `
WITH classified AS (
	SELECT
		id,
		CASE
			WHEN status = 'error' THEN 'error'
			WHEN status <> 'active' THEN 'disabled'
			WHEN rate_limit_reset_at IS NOT NULL AND rate_limit_reset_at > NOW() THEN 'rate_limited'
			WHEN overload_until IS NOT NULL AND overload_until > NOW() THEN 'overloaded'
			WHEN temp_unschedulable_until IS NOT NULL AND temp_unschedulable_until > NOW() THEN 'temp_unschedulable'
			WHEN schedulable = TRUE AND (
				auto_pause_on_expired = FALSE OR
				expires_at IS NULL OR
				expires_at > NOW()
			) THEN 'available'
			ELSE 'paused'
		END AS runtime_status
	FROM accounts
	WHERE deleted_at IS NULL
)
SELECT
	g.id,
	g.name,
	g.platform,
	COALESCE(g.description, '') AS description,
	COUNT(c.id) AS total_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'available') AS available_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'rate_limited') AS rate_limited_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'overloaded') AS overloaded_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'temp_unschedulable') AS temp_unschedulable_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'paused') AS paused_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'error') AS error_accounts,
	COUNT(c.id) FILTER (WHERE c.runtime_status = 'disabled') AS disabled_accounts
FROM groups g
LEFT JOIN account_groups ag ON ag.group_id = g.id
LEFT JOIN classified c ON c.id = ag.account_id
WHERE g.deleted_at IS NULL
  AND g.status = 'active'
GROUP BY g.id, g.name, g.platform, g.description
ORDER BY available_accounts DESC, total_accounts DESC, g.id ASC
`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query pool groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]PoolStatusGroup, 0)
	for rows.Next() {
		var item PoolStatusGroup
		if err := rows.Scan(
			&item.GroupID,
			&item.GroupName,
			&item.Platform,
			&item.Description,
			&item.TotalAccounts,
			&item.AvailableAccounts,
			&item.RateLimitedAccounts,
			&item.OverloadedAccounts,
			&item.TempUnschedulableAccount,
			&item.PausedAccounts,
			&item.ErrorAccounts,
			&item.DisabledAccounts,
		); err != nil {
			return nil, fmt.Errorf("scan pool group: %w", err)
		}
		item.AvailabilityRatio = calculateAvailabilityRatio(item.AvailableAccounts, item.TotalAccounts)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pool groups: %w", err)
	}

	return items, nil
}

func calculateAvailabilityRatio(available, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(available) / float64(total) * 100
}
