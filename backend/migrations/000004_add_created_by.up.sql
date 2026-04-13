ALTER TABLE tasks ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL;

UPDATE tasks SET created_by = (SELECT owner_id FROM projects WHERE projects.id = tasks.project_id);
