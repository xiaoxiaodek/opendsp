CREATE TABLE file_record (
    id           VARCHAR(32) PRIMARY KEY,
    namespace    VARCHAR(16) NOT NULL,
    storage_key  VARCHAR(512) NOT NULL,
    filename     VARCHAR(256),
    size         BIGINT DEFAULT 0,
    content_type VARCHAR(128),
    md5          VARCHAR(64),
    status       SMALLINT DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_file_record_status ON file_record(status, created_at);
