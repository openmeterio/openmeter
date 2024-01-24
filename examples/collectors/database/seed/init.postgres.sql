CREATE TABLE
    IF NOT EXISTS messages (
        message_id UUID PRIMARY KEY,
        account_id TEXT,
        sender TEXT,
        recipient TEXT,
        message TEXT,
        time TIMESTAMPTZ
    );
