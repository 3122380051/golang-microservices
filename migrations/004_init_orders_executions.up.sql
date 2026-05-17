CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    strategy_id UUID REFERENCES strategies(id) ON DELETE SET NULL,
    symbol_id UUID NOT NULL REFERENCES market_symbols(id),
    side TEXT NOT NULL,
    order_type TEXT NOT NULL,
    quantity NUMERIC(20, 8) NOT NULL,
    price NUMERIC(20, 8),
    status TEXT NOT NULL DEFAULT 'created',
    client_order_id TEXT NOT NULL UNIQUE,
    exchange_order_id TEXT,
    correlation_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    exchange TEXT NOT NULL,
    exchange_trade_id TEXT NOT NULL,
    fill_price NUMERIC(20, 8) NOT NULL,
    fill_quantity NUMERIC(20, 8) NOT NULL,
    fee NUMERIC(20, 8) NOT NULL DEFAULT 0,
    fee_asset TEXT NOT NULL DEFAULT '',
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (exchange, exchange_trade_id)
);

CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_strategy_id ON orders (strategy_id);
CREATE INDEX idx_orders_symbol_id ON orders (symbol_id);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_orders_created_at ON orders (created_at);
CREATE INDEX idx_orders_client_order_id ON orders (client_order_id);
CREATE INDEX idx_executions_order_id ON executions (order_id);
CREATE INDEX idx_executions_executed_at ON executions (executed_at);
