CREATE TABLE customers (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_number         TEXT NOT NULL UNIQUE,
    name                    TEXT NOT NULL,
    phone                   TEXT NOT NULL DEFAULT '',
    email                   TEXT NOT NULL DEFAULT '',
    notes                   TEXT NOT NULL DEFAULT '',
    loyalty_points_cached   INTEGER NOT NULL DEFAULT 0,
    visit_count_cached      INTEGER NOT NULL DEFAULT 0,
    lifetime_spend_cached   INTEGER NOT NULL DEFAULT 0,
    last_visit_at           TEXT,
    created_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_customers_name ON customers(name);
CREATE INDEX idx_customers_phone ON customers(phone);
CREATE INDEX idx_customers_email ON customers(email);

CREATE TABLE customer_addresses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    label       TEXT NOT NULL DEFAULT '',
    line1       TEXT NOT NULL DEFAULT '',
    line2       TEXT NOT NULL DEFAULT '',
    city        TEXT NOT NULL DEFAULT '',
    state       TEXT NOT NULL DEFAULT '',
    postal_code TEXT NOT NULL DEFAULT '',
    country     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE customer_notes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    note        TEXT NOT NULL,
    created_by  INTEGER REFERENCES users(id),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE customer_loyalty_transactions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    points      INTEGER NOT NULL, -- signed
    type        TEXT NOT NULL, -- EARN, REDEEM, ADJUST, EXPIRE, REFUND
    sale_id     INTEGER REFERENCES sales(id),
    reason      TEXT NOT NULL DEFAULT '',
    created_by  INTEGER REFERENCES users(id),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX idx_loyalty_tx_customer ON customer_loyalty_transactions(customer_id);
