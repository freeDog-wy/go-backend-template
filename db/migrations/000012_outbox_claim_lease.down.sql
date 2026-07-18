DROP INDEX IF EXISTS idx_outbox_events_claimable;

ALTER TABLE outbox_events
    DROP COLUMN claim_until,
    DROP COLUMN claimed_by;
