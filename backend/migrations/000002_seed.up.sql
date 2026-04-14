INSERT INTO users (id, name, email, password, created_at)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'Test User',
    'test@example.com',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj4J8F9.H2VC',
    now()
) ON CONFLICT (email) DO UPDATE SET name=EXCLUDED.name, password=EXCLUDED.password;

INSERT INTO projects (id, name, description, owner_id, created_at)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'Website Redesign',
    'Q2 initiative to overhaul the marketing site',
    'a0000000-0000-0000-0000-000000000001',
    now()
) ON CONFLICT DO NOTHING;

INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_at, updated_at)
VALUES
(
    'c0000000-0000-0000-0000-000000000001',
    'Audit current site performance',
    'Run Lighthouse and document scores across key pages',
    'done',
    'high',
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    now() + interval '2 days',
    now() - interval '5 days',
    now() - interval '2 days'
),
(
    'c0000000-0000-0000-0000-000000000002',
    'Design new component library',
    'Define tokens, typography scale, and base components in Figma',
    'in_progress',
    'high',
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    now() + interval '7 days',
    now() - interval '3 days',
    now() - interval '1 day'
),
(
    'c0000000-0000-0000-0000-000000000003',
    'Implement homepage redesign',
    'Build responsive homepage using the new design system',
    'todo',
    'medium',
    'b0000000-0000-0000-0000-000000000001',
    NULL,
    now() + interval '14 days',
    now(),
    now()
)
ON CONFLICT DO NOTHING;
