-- 销售/渠道子系统（纯增量，不修改现有 users/groups/user_subscriptions/api_keys 语义）

CREATE TABLE IF NOT EXISTS sales_partners (
    id              BIGSERIAL PRIMARY KEY,
    code            VARCHAR(64) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    auth_mode       VARCHAR(20) NOT NULL DEFAULT 'header',
    secret_hash     VARCHAR(128) NOT NULL,
    secret_hint     VARCHAR(16) NOT NULL DEFAULT '',
    rate_limit_rpm  INT NOT NULL DEFAULT 60,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_partners_status ON sales_partners(status);

CREATE TABLE IF NOT EXISTS sales_packages (
    id                  BIGSERIAL PRIMARY KEY,
    code                VARCHAR(64) NOT NULL UNIQUE,
    name                VARCHAR(100) NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    platform            VARCHAR(50) NOT NULL,
    group_id            BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    cycle_unit          VARCHAR(20) NOT NULL DEFAULT 'month',
    cycle_count         INT NOT NULL DEFAULT 1,
    validity_days       INT NOT NULL DEFAULT 30,
    key_policy          VARCHAR(32) NOT NULL DEFAULT 'reuse_current',
    auto_create_key     BOOLEAN NOT NULL DEFAULT TRUE,
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    store_visible       BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_packages_status ON sales_packages(status);
CREATE INDEX IF NOT EXISTS idx_sales_packages_platform ON sales_packages(platform);
CREATE INDEX IF NOT EXISTS idx_sales_packages_group_id ON sales_packages(group_id);
CREATE INDEX IF NOT EXISTS idx_sales_packages_store_visible ON sales_packages(store_visible);

CREATE TABLE IF NOT EXISTS sales_partner_packages (
    id                      BIGSERIAL PRIMARY KEY,
    partner_id              BIGINT NOT NULL REFERENCES sales_partners(id) ON DELETE CASCADE,
    package_id              BIGINT NOT NULL REFERENCES sales_packages(id) ON DELETE CASCADE,
    external_package_code   VARCHAR(100) NOT NULL,
    external_package_name   VARCHAR(150) NOT NULL DEFAULT '',
    sale_price              DECIMAL(20, 8) NOT NULL DEFAULT 0,
    currency                VARCHAR(16) NOT NULL DEFAULT 'CNY',
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(partner_id, external_package_code),
    UNIQUE(partner_id, package_id)
);

CREATE INDEX IF NOT EXISTS idx_sales_partner_packages_partner_id ON sales_partner_packages(partner_id);
CREATE INDEX IF NOT EXISTS idx_sales_partner_packages_package_id ON sales_partner_packages(package_id);
CREATE INDEX IF NOT EXISTS idx_sales_partner_packages_status ON sales_partner_packages(status);

CREATE TABLE IF NOT EXISTS sales_user_bindings (
    id                  BIGSERIAL PRIMARY KEY,
    partner_id          BIGINT NOT NULL REFERENCES sales_partners(id) ON DELETE CASCADE,
    external_user_id    VARCHAR(100) NOT NULL,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    external_email      VARCHAR(255) NOT NULL DEFAULT '',
    external_name       VARCHAR(100) NOT NULL DEFAULT '',
    metadata            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(partner_id, external_user_id)
);

CREATE INDEX IF NOT EXISTS idx_sales_user_bindings_user_id ON sales_user_bindings(user_id);
CREATE INDEX IF NOT EXISTS idx_sales_user_bindings_partner_id ON sales_user_bindings(partner_id);

CREATE TABLE IF NOT EXISTS sales_orders (
    id                      BIGSERIAL PRIMARY KEY,
    partner_id              BIGINT NOT NULL REFERENCES sales_partners(id) ON DELETE RESTRICT,
    external_order_id       VARCHAR(100) NOT NULL,
    external_user_id        VARCHAR(100) NOT NULL,
    user_id                 BIGINT REFERENCES users(id) ON DELETE SET NULL,
    package_id              BIGINT NOT NULL REFERENCES sales_packages(id) ON DELETE RESTRICT,
    order_type              VARCHAR(20) NOT NULL DEFAULT 'purchase',
    status                  VARCHAR(32) NOT NULL DEFAULT 'pending',
    subscription_id         BIGINT REFERENCES user_subscriptions(id) ON DELETE SET NULL,
    api_key_id              BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    amount                  DECIMAL(20, 8) NOT NULL DEFAULT 0,
    currency                VARCHAR(16) NOT NULL DEFAULT 'CNY',
    package_snapshot        JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_payload             JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_snapshot         JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message           TEXT NOT NULL DEFAULT '',
    fulfilled_at            TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(partner_id, external_order_id)
);

CREATE INDEX IF NOT EXISTS idx_sales_orders_partner_id ON sales_orders(partner_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_package_id ON sales_orders(package_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_user_id ON sales_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_status ON sales_orders(status);
CREATE INDEX IF NOT EXISTS idx_sales_orders_created_at ON sales_orders(created_at DESC);
