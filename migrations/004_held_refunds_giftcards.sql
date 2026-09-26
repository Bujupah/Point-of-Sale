CREATE TABLE held_sales (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    register_id     INTEGER NOT NULL REFERENCES registers(id),
    cashier_id      INTEGER NOT NULL REFERENCES users(id),
    customer_id     INTEGER REFERENCES customers(id),
    note            TEXT NOT NULL DEFAULT '',
    table_name      TEXT NOT NULL DEFAULT '',
    ticket_name     TEXT NOT NULL DEFAULT '',
    subtotal        INTEGER NOT NULL DEFAULT 0,
    discount_total  INTEGER NOT NULL DEFAULT 0,
    tax_total       INTEGER NOT NULL DEFAULT 0,
    total           INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_held_sales_register ON held_sales(register_id);

CREATE TABLE held_sale_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    held_sale_id    INTEGER NOT NULL REFERENCES held_sales(id) ON DELETE CASCADE,
    product_id      INTEGER REFERENCES products(id),
    name_snapshot   TEXT NOT NULL,
    variant_snapshot TEXT NOT NULL DEFAULT '',
    quantity        INTEGER NOT NULL,
    unit_price      INTEGER NOT NULL,
    notes           TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_held_sale_items_held_sale ON held_sale_items(held_sale_id);

CREATE TABLE refunds (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    sale_id     INTEGER NOT NULL REFERENCES sales(id),
    shift_id    INTEGER NOT NULL REFERENCES shifts(id),
    cashier_id  INTEGER NOT NULL REFERENCES users(id),
    approver_id INTEGER REFERENCES users(id),
    reason      TEXT NOT NULL DEFAULT '',
    total       INTEGER NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_refunds_sale ON refunds(sale_id);
CREATE INDEX idx_refunds_shift ON refunds(shift_id);

CREATE TABLE refund_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    refund_id       INTEGER NOT NULL REFERENCES refunds(id) ON DELETE CASCADE,
    sale_item_id    INTEGER NOT NULL REFERENCES sale_items(id),
    quantity        INTEGER NOT NULL,
    amount          INTEGER NOT NULL
);

CREATE TABLE refund_payments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    refund_id   INTEGER NOT NULL REFERENCES refunds(id) ON DELETE CASCADE,
    method      TEXT NOT NULL,
    amount      INTEGER NOT NULL,
    reference   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE gift_cards (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    code            TEXT NOT NULL UNIQUE,
    initial_value   INTEGER NOT NULL,
    balance         INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, DEPLETED, DISABLED, EXPIRED
    expires_at      TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE gift_card_transactions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    gift_card_id    INTEGER NOT NULL REFERENCES gift_cards(id),
    type            TEXT NOT NULL, -- ISSUE, REDEEM, ADJUST, REFUND
    amount          INTEGER NOT NULL, -- signed
    sale_id         INTEGER REFERENCES sales(id),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE printers (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    name                    TEXT NOT NULL,
    type                    TEXT NOT NULL DEFAULT 'RECEIPT', -- RECEIPT, KITCHEN
    connection              TEXT NOT NULL DEFAULT 'SIMULATED', -- SIMULATED, WINDOWS_SPOOL, COM, LPT
    config_json             TEXT NOT NULL DEFAULT '{}',
    paper_width_mm          INTEGER NOT NULL DEFAULT 80,
    is_default_receipt      INTEGER NOT NULL DEFAULT 0,
    is_default_kitchen      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE devices (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL, -- SCANNER, SCALE, DRAWER, CUSTOMER_DISPLAY, PAYMENT_TERMINAL
    config_json TEXT NOT NULL DEFAULT '{}'
);
