-- Add order column to tasks table for supporting task reordering within columns
ALTER TABLE tasks ADD COLUMN "order" INT DEFAULT 0;

-- Populate existing tasks with sequential order based on created_at
WITH ranked_tasks AS (
  SELECT 
    id, 
    project_id,
    status,
    ROW_NUMBER() OVER (PARTITION BY project_id, status ORDER BY created_at ASC) as seq_order
  FROM tasks
)
UPDATE tasks SET "order" = ranked_tasks.seq_order
FROM ranked_tasks
WHERE tasks.id = ranked_tasks.id;

-- Create index for efficient querying by order
CREATE INDEX IF NOT EXISTS idx_tasks_order ON tasks(project_id, status, "order");
