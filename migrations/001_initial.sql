-- Core: settings, locations, registers, RBAC, catalog basics, stock, shifts, cash, sales, payments, audit, sync.
-- Money columns are always INTEGER minor units (see internal/domain.Money). Never REAL for currency.

CREATE TABLE settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE locations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    address     TEXT NOT NULL DEFAULT '',
    tax_id      TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE registers (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    location_id INTEGER NOT NULL REFERENCES locations(id),
    name        TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE roles (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name    TEXT NOT NULL UNIQUE
);

CREATE TABLE permissions (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    code    TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
    role_id       INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    username    TEXT NOT NULL UNIQUE,
    pin_hash    TEXT NOT NULL,
    role_id     INTEGER NOT NULL REFERENCES roles(id),
    active      INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    register_id INTEGER REFERENCES registers(id),
    locked      INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    expires_at  TEXT NOT NULL
);

CREATE TABLE tax_rates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    rate_bps    INTEGER NOT NULL, -- basis points, e.g. 1000 = 10.00%
    inclusive   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE categories (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id           INTEGER REFERENCES categories(id),
    name                TEXT NOT NULL,
    name_translations   TEXT NOT NULL DEFAULT '{}', -- JSON: {"es": "...", "ar": "..."}
    sort_order          INTEGER NOT NULL DEFAULT 0,
    active              INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE products (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sku                 TEXT NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    name_translations   TEXT NOT NULL DEFAULT '{}',
    description         TEXT NOT NULL DEFAULT '',
    category_id         INTEGER REFERENCES categories(id),
    price               INTEGER NOT NULL, -- minor units
    cost                INTEGER NOT NULL DEFAULT 0,
    tax_rate_id         INTEGER REFERENCES tax_rates(id),
    track_stock         INTEGER NOT NULL DEFAULT 1,
    reorder_level       INTEGER NOT NULL DEFAULT 0,
    image_thumbnail     TEXT,
    image_medium        TEXT,
    badge               TEXT NOT NULL DEFAULT '', -- SALE, NEW, '' etc (computed badges like LOW_STOCK are derived, not stored)
    original_price      INTEGER, -- non-null implies promotion vs price
    is_open_item        INTEGER NOT NULL DEFAULT 0,
    active              INTEGER NOT NULL DEFAULT 1,
    favorite            INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_active ON products(active);

CREATE TABLE product_barcodes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id  INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    barcode     TEXT NOT NULL UNIQUE
);
CREATE INDEX idx_product_barcodes_product ON product_barcodes(product_id);

CREATE TABLE stock_movements (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id      INTEGER NOT NULL REFERENCES products(id),
    type            TEXT NOT NULL, -- INITIAL, PURCHASE, SALE, RETURN, ADJUSTMENT, DAMAGE, TRANSFER_IN, TRANSFER_OUT
    quantity        INTEGER NOT NULL, -- signed delta
    reference_type  TEXT NOT NULL DEFAULT '', -- e.g. 'sale', 'refund', 'manual'
    reference_id    INTEGER,
    note            TEXT NOT NULL DEFAULT '',
    created_by      INTEGER REFERENCES users(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_stock_movements_product ON stock_movements(product_id);

CREATE TABLE inventory_balances (
    product_id  INTEGER PRIMARY KEY REFERENCES products(id),
    quantity    INTEGER NOT NULL DEFAULT 0,
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE shifts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    register_id     INTEGER NOT NULL REFERENCES registers(id),
    cashier_id      INTEGER NOT NULL REFERENCES users(id),
    opening_float   INTEGER NOT NULL DEFAULT 0,
    opening_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    closing_at      TEXT,
    status          TEXT NOT NULL DEFAULT 'OPEN', -- OPEN, CLOSING, CLOSED
    expected_cash   INTEGER,
    counted_cash    INTEGER,
    difference      INTEGER,
    notes           TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_shifts_register ON shifts(register_id);
CREATE INDEX idx_shifts_status ON shifts(status);

CREATE TABLE cash_movements (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    shift_id    INTEGER NOT NULL REFERENCES shifts(id),
    type        TEXT NOT NULL, -- CASH_IN, CASH_OUT, CASH_DROP, FLOAT_ADJUSTMENT, SALE_CASH, REFUND_CASH
    amount      INTEGER NOT NULL, -- signed, minor units
    reason      TEXT NOT NULL DEFAULT '',
    cashier_id  INTEGER NOT NULL REFERENCES users(id),
    reference_type TEXT NOT NULL DEFAULT '',
    reference_id   INTEGER,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_cash_movements_shift ON cash_movements(shift_id);

CREATE TABLE sales (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    receipt_number  TEXT NOT NULL UNIQUE,
    register_id     INTEGER NOT NULL REFERENCES registers(id),
    shift_id        INTEGER NOT NULL REFERENCES shifts(id),
    cashier_id      INTEGER NOT NULL REFERENCES users(id),
    customer_id     INTEGER REFERENCES customers(id),
    subtotal        INTEGER NOT NULL,
    discount_total  INTEGER NOT NULL DEFAULT 0,
    discount_type   TEXT NOT NULL DEFAULT '', -- FIXED, PERCENT, ''
    discount_reason TEXT NOT NULL DEFAULT '',
    discount_approved_by INTEGER REFERENCES users(id),
    tax_total       INTEGER NOT NULL DEFAULT 0,
    total           INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'COMPLETED', -- COMPLETED, VOIDED, REFUNDED, PARTIALLY_REFUNDED
    note            TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_sales_shift ON sales(shift_id);
CREATE INDEX idx_sales_customer ON sales(customer_id);
CREATE INDEX idx_sales_created_at ON sales(created_at);

CREATE TABLE sale_items (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sale_id             INTEGER NOT NULL REFERENCES sales(id),
    product_id          INTEGER REFERENCES products(id),
    sku_snapshot        TEXT NOT NULL DEFAULT '',
    barcode_snapshot    TEXT NOT NULL DEFAULT '',
    name_snapshot       TEXT NOT NULL,
    variant_snapshot    TEXT NOT NULL DEFAULT '', -- JSON
    quantity            INTEGER NOT NULL,
    unit_price          INTEGER NOT NULL,
    cost_snapshot       INTEGER NOT NULL DEFAULT 0,
    discount_amount     INTEGER NOT NULL DEFAULT 0,
    discount_type       TEXT NOT NULL DEFAULT '',
    tax_amount          INTEGER NOT NULL DEFAULT 0,
    line_total          INTEGER NOT NULL,
    notes               TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_sale_items_sale ON sale_items(sale_id);

CREATE TABLE sale_item_modifiers (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sale_item_id        INTEGER NOT NULL REFERENCES sale_items(id),
    name_snapshot       TEXT NOT NULL,
    price_adjustment    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_sale_item_modifiers_item ON sale_item_modifiers(sale_item_id);

CREATE TABLE payments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    sale_id     INTEGER NOT NULL REFERENCES sales(id),
    method      TEXT NOT NULL, -- CASH, CARD, BANK_TRANSFER, GIFT_CARD, OTHER, MOBILE, VOUCHER, CHEQUE, CUSTOM
    amount      INTEGER NOT NULL,
    tendered    INTEGER NOT NULL DEFAULT 0,
    change_due  INTEGER NOT NULL DEFAULT 0,
    reference   TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'APPROVED', -- APPROVED, DECLINED, CANCELLED, PENDING
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_payments_sale ON payments(sale_id);

CREATE TABLE audit_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    event           TEXT NOT NULL,
    user_id         INTEGER REFERENCES users(id),
    approver_id     INTEGER REFERENCES users(id),
    entity_type     TEXT NOT NULL DEFAULT '',
    entity_id       INTEGER,
    details_json    TEXT NOT NULL DEFAULT '{}',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_audit_logs_event ON audit_logs(event);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);

CREATE TABLE sync_queue (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type     TEXT NOT NULL,
    entity_id       INTEGER NOT NULL,
    operation       TEXT NOT NULL,
    payload_json    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING', -- PENDING, SYNCING, SYNCED, ERROR
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_attempt_at TEXT,
    synced_at       TEXT
);
