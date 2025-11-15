-- +goose Up
CREATE TABLE pull_requests (
                               pull_request_id VARCHAR(255) PRIMARY KEY,
                               pull_request_name VARCHAR(500) NOT NULL,
                               author_id VARCHAR(255) NOT NULL REFERENCES users(user_id),
                               status VARCHAR(20) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
                               created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                               merged_at TIMESTAMP
);

CREATE INDEX idx_pull_requests_author_id ON pull_requests(author_id);
CREATE INDEX idx_pull_requests_status ON pull_requests(status);
CREATE INDEX idx_pull_requests_created_at ON pull_requests(created_at);

-- +goose Down
DROP TABLE pull_requests;