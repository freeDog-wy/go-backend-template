ALTER TABLE outbox_events
    ADD COLUMN claimed_by VARCHAR(36) NOT NULL DEFAULT '',
    ADD COLUMN claim_until TIMESTAMPTZ;

CREATE INDEX idx_outbox_events_claimable
    ON outbox_events (claim_until, id)
    WHERE published_at IS NULL;
