ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_actor_id_fkey;
ALTER TABLE audit_logs
    ADD CONSTRAINT audit_logs_actor_id_fkey
    FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE system_settings DROP CONSTRAINT IF EXISTS system_settings_updated_by_fkey;
ALTER TABLE system_settings
    ADD CONSTRAINT system_settings_updated_by_fkey
    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
