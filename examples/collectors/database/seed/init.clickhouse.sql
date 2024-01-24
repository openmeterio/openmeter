CREATE TABLE
    IF NOT EXISTS messages (
        message_id UUID,
        account_id String,
        sender String,
        recipient String,
        message String,
        time DateTime('UTC')
    ) ENGINE = MergeTree() PRIMARY KEY (message_id);
