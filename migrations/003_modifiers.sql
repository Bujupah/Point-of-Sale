-- Product variants (e.g. Individual / For 2 / Family XL) and modifier groups
-- (e.g. Cooking level, Extras) shared across products.

CREATE TABLE product_variant_groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    product_id  INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    required    INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_variant_groups_product ON product_variant_groups(product_id);

CREATE TABLE product_variants (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id            INTEGER NOT NULL REFERENCES product_variant_groups(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    price_adjustment    INTEGER NOT NULL DEFAULT 0,
    sku_suffix          TEXT NOT NULL DEFAULT '',
    sort_order          INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_variants_group ON product_variants(group_id);

CREATE TABLE modifier_groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    min_select  INTEGER NOT NULL DEFAULT 0,
    max_select  INTEGER NOT NULL DEFAULT 1,
    required    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE modifier_options (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id            INTEGER NOT NULL REFERENCES modifier_groups(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    price_adjustment    INTEGER NOT NULL DEFAULT 0,
    sort_order          INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_modifier_options_group ON modifier_options(group_id);

CREATE TABLE product_modifier_groups (
    product_id          INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    modifier_group_id   INTEGER NOT NULL REFERENCES modifier_groups(id) ON DELETE CASCADE,
    sort_order          INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (product_id, modifier_group_id)
);
