-- USERS
CREATE TABLE IF NOT EXISTS users (
  id               BIGSERIAL PRIMARY KEY,
  email            TEXT UNIQUE NOT NULL,
  username         TEXT UNIQUE NOT NULL,
  password_hash    TEXT NOT NULL,
  deposit_balance  NUMERIC(18,2) NOT NULL DEFAULT 0,
  role             TEXT NOT NULL DEFAULT 'user',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- BOOKS
CREATE TABLE IF NOT EXISTS books (
  id          BIGSERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  category    TEXT NOT NULL,
  rental_cost NUMERIC(18,2) NOT NULL CHECK (rental_cost >= 0),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- STOCK per copy
DO $$ BEGIN
  CREATE TYPE book_item_status AS ENUM ('AVAILABLE','BOOKED','RENTED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS book_items (
  id           BIGSERIAL PRIMARY KEY,
  book_id      BIGINT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  status       book_item_status NOT NULL DEFAULT 'AVAILABLE',
  booked_until TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_book_items_book_available
  ON book_items(book_id) WHERE status = 'AVAILABLE';
CREATE INDEX IF NOT EXISTS idx_book_items_book_status
  ON book_items(book_id, status);

-- RENTALS
DO $$ BEGIN
  CREATE TYPE rental_status AS ENUM ('BOOKED','PAID','ACTIVE','RETURNED','CANCELED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS rentals (
  id                 BIGSERIAL PRIMARY KEY,
  user_id            BIGINT NOT NULL REFERENCES users(id),
  book_id            BIGINT NOT NULL REFERENCES books(id),
  book_item_id       BIGINT NOT NULL REFERENCES book_items(id),
  status             rental_status NOT NULL DEFAULT 'BOOKED',
  rental_cost        NUMERIC(18,2) NOT NULL CHECK (rental_cost >= 0),
  booked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_due_at     TIMESTAMPTZ NOT NULL,
  paid_at            TIMESTAMPTZ,
  activated_at       TIMESTAMPTZ,
  returned_at        TIMESTAMPTZ,
  canceled_at        TIMESTAMPTZ,
  xendit_invoice_id  TEXT
);
CREATE INDEX IF NOT EXISTS idx_rentals_user ON rentals(user_id);
CREATE INDEX IF NOT EXISTS idx_rentals_status ON rentals(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_active_copy
  ON rentals(book_item_id) WHERE status IN ('BOOKED','PAID','ACTIVE');

-- WALLET TOPUPS & LEDGER
DO $$ BEGIN
  CREATE TYPE topup_status AS ENUM ('PENDING','PAID','EXPIRED','FAILED');
  CREATE TYPE ledger_type AS ENUM ('TOPUP_CONFIRMED','RENTAL_CHARGE','RENTAL_REFUND','ADJUSTMENT');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS wallet_topups (
  id                BIGSERIAL PRIMARY KEY,
  user_id           BIGINT NOT NULL REFERENCES users(id),
  amount            NUMERIC(18,2) NOT NULL CHECK (amount > 0),
  status            topup_status NOT NULL DEFAULT 'PENDING',
  xendit_invoice_id TEXT UNIQUE,
  payment_link      TEXT,
  expires_at        TIMESTAMPTZ,
  paid_at           TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id),
  ref_table     TEXT,
  ref_id        BIGINT,
  entry_type    ledger_type NOT NULL,
  amount        NUMERIC(18,2) NOT NULL,
  balance_after NUMERIC(18,2) NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ledger_user ON wallet_ledger(user_id);

-- USERS
CREATE TABLE IF NOT EXISTS users (
  id               BIGSERIAL PRIMARY KEY,
  email            TEXT UNIQUE NOT NULL,
  username         TEXT UNIQUE NOT NULL,
  password_hash    TEXT NOT NULL,
  deposit_balance  NUMERIC(18,2) NOT NULL DEFAULT 0,
  role             TEXT NOT NULL DEFAULT 'user',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- BOOKS
CREATE TABLE IF NOT EXISTS books (
  id          BIGSERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  category    TEXT NOT NULL,
  rental_cost NUMERIC(18,2) NOT NULL CHECK (rental_cost >= 0),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- STOCK per copy
DO $$ BEGIN
  CREATE TYPE book_item_status AS ENUM ('AVAILABLE','BOOKED','RENTED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS book_items (
  id           BIGSERIAL PRIMARY KEY,
  book_id      BIGINT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
  status       book_item_status NOT NULL DEFAULT 'AVAILABLE',
  booked_until TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_book_items_book_available
  ON book_items(book_id) WHERE status = 'AVAILABLE';
CREATE INDEX IF NOT EXISTS idx_book_items_book_status
  ON book_items(book_id, status);

-- RENTALS
DO $$ BEGIN
  CREATE TYPE rental_status AS ENUM ('BOOKED','PAID','ACTIVE','RETURNED','CANCELED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS rentals (
  id                 BIGSERIAL PRIMARY KEY,
  user_id            BIGINT NOT NULL REFERENCES users(id),
  book_id            BIGINT NOT NULL REFERENCES books(id),
  book_item_id       BIGINT NOT NULL REFERENCES book_items(id),
  status             rental_status NOT NULL DEFAULT 'BOOKED',
  rental_cost        NUMERIC(18,2) NOT NULL CHECK (rental_cost >= 0),
  booked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  payment_due_at     TIMESTAMPTZ NOT NULL,
  paid_at            TIMESTAMPTZ,
  activated_at       TIMESTAMPTZ,
  returned_at        TIMESTAMPTZ,
  canceled_at        TIMESTAMPTZ,
  xendit_invoice_id  TEXT
);
CREATE INDEX IF NOT EXISTS idx_rentals_user ON rentals(user_id);
CREATE INDEX IF NOT EXISTS idx_rentals_status ON rentals(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_active_copy
  ON rentals(book_item_id) WHERE status IN ('BOOKED','PAID','ACTIVE');

-- WALLET TOPUPS & LEDGER
DO $$ BEGIN
  CREATE TYPE topup_status AS ENUM ('PENDING','PAID','EXPIRED','FAILED');
  CREATE TYPE ledger_type AS ENUM ('TOPUP_CONFIRMED','RENTAL_CHARGE','RENTAL_REFUND','ADJUSTMENT');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS wallet_topups (
  id                BIGSERIAL PRIMARY KEY,
  user_id           BIGINT NOT NULL REFERENCES users(id),
  amount            NUMERIC(18,2) NOT NULL CHECK (amount > 0),
  status            topup_status NOT NULL DEFAULT 'PENDING',
  xendit_invoice_id TEXT UNIQUE,
  payment_link      TEXT,
  expires_at        TIMESTAMPTZ,
  paid_at           TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id),
  ref_table     TEXT,
  ref_id        BIGINT,
  entry_type    ledger_type NOT NULL,
  amount        NUMERIC(18,2) NOT NULL,
  balance_after NUMERIC(18,2) NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ledger_user ON wallet_ledger(user_id);

CREATE OR REPLACE VIEW v_books_with_availability AS
SELECT
  b.id, b.name, b.category, b.rental_cost,
  COUNT(*) FILTER (WHERE bi.status='AVAILABLE')::BIGINT AS stock_availability
FROM books b
LEFT JOIN book_items bi ON bi.book_id=b.id
GROUP BY b.id, b.name, b.category, b.rental_cost;
