DROP TABLE IF EXISTS reminders;
DROP TABLE IF EXISTS reminder_templates;
ALTER TABLE documents DROP COLUMN IF EXISTS last_reminder_at, DROP COLUMN IF EXISTS reminder_count;
