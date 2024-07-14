CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    active BOOLEAN NOT NULL,
    last_active DATETIME NOT NULL,
    review_collected BOOLEAN NOT NULL,
    session_data JSON
);
