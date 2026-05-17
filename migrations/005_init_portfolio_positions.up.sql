CREATE TABLE portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    total_equity NUMERIC(20, 8) NOT NULL DEFAULT 0,
    available_balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
    used_margin NUMERIC(20, 8) NOT NULL DEFAULT 0,
    unrealized_pnl NUMERIC(20, 8) NOT NULL DEFAULT 0,
    realized_pnl NUMERIC(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    symbol_id UUID NOT NULL REFERENCES market_symbols(id),
    side TEXT NOT NULL,
    quantity NUMERIC(20, 8) NOT NULL DEFAULT 0,
    entry_price NUMERIC(20, 8) NOT NULL DEFAULT 0,
    mark_price NUMERIC(20, 8) NOT NULL DEFAULT 0,
    unrealized_pnl NUMERIC(20, 8) NOT NULL DEFAULT 0,
    leverage NUMERIC(10, 2) NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (portfolio_id, symbol_id, side)
);

CREATE INDEX idx_portfolios_user_id ON portfolios (user_id);
CREATE INDEX idx_portfolios_updated_at ON portfolios (updated_at);
CREATE INDEX idx_positions_portfolio_id ON positions (portfolio_id);
CREATE INDEX idx_positions_symbol_id ON positions (symbol_id);
CREATE INDEX idx_positions_updated_at ON positions (updated_at);
