CREATE TABLE IF NOT EXISTS cash_box_history (
    id UUID NOT NULL PRIMARY KEY,
    cash_box_session_id UUID REFERENCES cash_box_sessions(id),
    action_type VARCHAR(55),
    amount NUMERIC(10, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);