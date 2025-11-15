-- +goose Up
CREATE TABLE auth_tokens (
                             id SERIAL PRIMARY KEY,
                             user_id VARCHAR(255) NOT NULL,
                             token TEXT NOT NULL UNIQUE,
                             expires_at TIMESTAMP NOT NULL,
                             created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_tokens_user_id ON auth_tokens(user_id);
CREATE INDEX idx_auth_tokens_token ON auth_tokens(token);
CREATE INDEX idx_auth_tokens_expires_at ON auth_tokens(expires_at);

-- +goose Down
DROP TABLE auth_tokens;