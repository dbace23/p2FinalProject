-- USERS
CREATE TABLE IF NOT EXISTS users (
  id               BIGSERIAL PRIMARY KEY,
  email            TEXT UNIQUE NOT NULL,
  username         TEXT UNIQUE NOT NULL,
  password_hash    TEXT NOT NULL,
  deposit_balance  NUMERIC(18,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- BOOKS  
CREATE TABLE IF NOT EXISTS books (
  id             BIGSERIAL PRIMARY KEY,
  name           TEXT NOT NULL,
  category       TEXT NOT NULL,
  rental_cost    NUMERIC(18,2) NOT NULL CHECK (rental_cost >= 0),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- STOCK by physical copy 
CREATE TYPE book_item_status AS ENUM ('AVAILABLE','BOOKED','RENTED');

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
CREATE TYPE rental_status AS ENUM ('BOOKED','PAID','ACTIVE','RETURNED','CANCELED');

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
  xendit_invoice_id  TEXT,       
  notes              TEXT
);

CREATE INDEX IF NOT EXISTS idx_rentals_user ON rentals(user_id);
CREATE INDEX IF NOT EXISTS idx_rentals_status ON rentals(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_active_copy
  ON rentals(book_item_id)
  WHERE status IN ('BOOKED','PAID','ACTIVE');

-- WALLET: TOPUPS via Xendit
CREATE TYPE topup_status AS ENUM ('PENDING','PAID','EXPIRED','FAILED');

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

-- WALLET: LEDGER  
-- positive amount increases balance (CREDIT), negative decreases (DEBIT)
CREATE TYPE ledger_type AS ENUM ('TOPUP_CONFIRMED','RENTAL_CHARGE','RENTAL_REFUND','ADJUSTMENT');

CREATE TABLE IF NOT EXISTS wallet_ledger (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users(id),
  ref_table     TEXT,          -- 'wallet_topups' | 'rentals'
  ref_id        BIGINT,
  entry_type    ledger_type NOT NULL,
  amount        NUMERIC(18,2) NOT NULL,   -- +credit / -debit
  balance_after NUMERIC(18,2) NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_user ON wallet_ledger(user_id);

 