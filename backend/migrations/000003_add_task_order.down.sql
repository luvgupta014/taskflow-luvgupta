-- Rollback: Remove order column and index
DROP INDEX IF EXISTS idx_tasks_order;
ALTER TABLE tasks DROP COLUMN IF EXISTS "order";
