-- +goose Up
CREATE TABLE teams (
                       team_name VARCHAR(255) PRIMARY KEY,
                       created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE teams;