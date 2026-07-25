ALTER TABLE logs RENAME TO audit_logs;

ALTER INDEX idx_logs_actor_user_id RENAME TO idx_audit_logs_actor_user_id;
ALTER INDEX idx_logs_target_type RENAME TO idx_audit_logs_target_type;
ALTER INDEX idx_logs_target_id RENAME TO idx_audit_logs_target_id;
ALTER INDEX idx_logs_action RENAME TO idx_audit_logs_action;
ALTER INDEX idx_logs_result RENAME TO idx_audit_logs_result;
ALTER INDEX idx_logs_trace_id RENAME TO idx_audit_logs_trace_id;
