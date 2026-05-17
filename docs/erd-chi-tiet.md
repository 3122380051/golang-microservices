# ERD chi tiết cho hệ thống trading crypto

## Mục tiêu
ERD này mô tả các thực thể chính, khóa chính, khóa ngoại và quan hệ giữa các bảng phục vụ hệ thống microservice trading crypto bằng Go.

## ERD

```mermaid
erDiagram
    USERS {
        uuid id PK
        string email
        string password_hash
        string full_name
        string status
        timestamp created_at
        timestamp updated_at
    }

    ROLES {
        uuid id PK
        string name
        string description
        timestamp created_at
    }

    USER_ROLES {
        uuid user_id FK
        uuid role_id FK
    }

    STRATEGIES {
        uuid id PK
        uuid user_id FK
        string name
        string type
        json config_json
        string status
        timestamp created_at
        timestamp updated_at
    }

    MARKET_SYMBOLS {
        uuid id PK
        string exchange
        string symbol
        string base_asset
        string quote_asset
        string status
    }

    ORDERS {
        uuid id PK
        uuid user_id FK
        uuid strategy_id FK
        uuid symbol_id FK
        string side
        string order_type
        decimal quantity
        decimal price
        string status
        string client_order_id
        string exchange_order_id
        string correlation_id
        timestamp created_at
        timestamp updated_at
    }

    EXECUTIONS {
        uuid id PK
        uuid order_id FK
        string exchange
        string exchange_trade_id
        decimal fill_price
        decimal fill_quantity
        decimal fee
        string fee_asset
        timestamp executed_at
    }

    PORTFOLIOS {
        uuid id PK
        uuid user_id FK
        decimal total_equity
        decimal available_balance
        decimal used_margin
        decimal unrealized_pnl
        decimal realized_pnl
        timestamp updated_at
    }

    POSITIONS {
        uuid id PK
        uuid portfolio_id FK
        uuid symbol_id FK
        string side
        decimal quantity
        decimal entry_price
        decimal mark_price
        decimal unrealized_pnl
        decimal leverage
        timestamp updated_at
    }

    AUDIT_LOGS {
        uuid id PK
        uuid user_id FK
        string action
        string entity_type
        uuid entity_id
        json metadata_json
        string trace_id
        timestamp created_at
    }

    NOTIFICATIONS {
        uuid id PK
        uuid user_id FK
        string type
        string channel
        string title
        string message
        string status
        timestamp sent_at
    }

    USERS ||--o{ USER_ROLES : has
    ROLES ||--o{ USER_ROLES : assigned_to
    USERS ||--o{ STRATEGIES : owns
    USERS ||--o{ ORDERS : places
    STRATEGIES ||--o{ ORDERS : generates
    MARKET_SYMBOLS ||--o{ ORDERS : trades
    ORDERS ||--o{ EXECUTIONS : produces
    USERS ||--o{ PORTFOLIOS : owns
    PORTFOLIOS ||--o{ POSITIONS : contains
    MARKET_SYMBOLS ||--o{ POSITIONS : tracks
    USERS ||--o{ AUDIT_LOGS : triggers
    USERS ||--o{ NOTIFICATIONS : receives
```

## Ghi chú quan hệ
- USERS liên kết với ROLES qua USER_ROLES để hỗ trợ phân quyền nhiều-nhiều.
- USERS sở hữu STRATEGIES, ORDERS, PORTFOLIOS, AUDIT_LOGS và NOTIFICATIONS.
- STRATEGIES sinh ra ORDERS khi chiến lược tạo tín hiệu giao dịch.
- ORDERS có thể có nhiều EXECUTIONS nếu khớp nhiều lần.
- PORTFOLIOS chứa POSITIONS theo từng MARKET_SYMBOLS.
- AUDIT_LOGS giữ trace cho các thao tác nhạy cảm và có thể gắn với entity bất kỳ qua entity_type và entity_id.
