-- +goose Up
CREATE TABLE users (
                       user_id VARCHAR(255) PRIMARY KEY,
                       username VARCHAR(255) NOT NULL,
                       team_name VARCHAR(255) NOT NULL REFERENCES teams(team_name) ON DELETE CASCADE,
                       is_active BOOLEAN NOT NULL DEFAULT true,
                       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                       updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_team_name ON users(team_name);
CREATE INDEX idx_users_is_active ON users(is_active);
CREATE INDEX idx_users_team_active ON users(team_name, is_active);

-- +goose Down
DROP TABLE users;