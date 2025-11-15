-- +goose Up
CREATE TABLE pr_reviewers (
                              id SERIAL PRIMARY KEY,
                              pull_request_id VARCHAR(255) NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
                              user_id VARCHAR(255) NOT NULL REFERENCES users(user_id),
                              assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
                              UNIQUE(pull_request_id, user_id)
);

CREATE INDEX idx_pr_reviewers_pull_request_id ON pr_reviewers(pull_request_id);
CREATE INDEX idx_pr_reviewers_user_id ON pr_reviewers(user_id);

-- +goose Down
DROP TABLE pr_reviewers;