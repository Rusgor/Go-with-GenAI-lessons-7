CREATE TABLE loans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    copy_id BIGINT NOT NULL REFERENCES copies(id) ON DELETE RESTRICT,
    checked_out_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ NOT NULL,
    returned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_loans_due_after_checkout CHECK (due_at > checked_out_at)
);
CREATE INDEX idx_loans_user_id ON loans (user_id, id DESC);
CREATE INDEX idx_loans_copy_id ON loans (copy_id);
CREATE UNIQUE INDEX idx_loans_one_active_per_copy ON loans (copy_id) WHERE returned_at IS NULL;
