# TaskFlow

A task management app where you can create projects, break them into tasks, assign them to teammates, and track progress on a Kanban board.

---

## 1. Overview

**Stack**

| Layer | Technology |
|---|---|
| Backend | Go 1.22, chi, pgx/v5, golang-migrate |
| Frontend | React 18, TypeScript, Vite, TailwindCSS, Radix UI, @dnd-kit |
| Database | PostgreSQL 15 |
| Infrastructure | Docker Compose, multi-stage Dockerfiles |

What's built:
- Register and login with JWT auth (bcrypt cost 12, 24h token)
- Create and manage projects
- Add tasks with status, priority, assignee, and due date
- Kanban board with drag-and-drop between columns and reordering within columns
- List view with status and assignee filters
- Stats page showing task counts by status and by assignee
- Rate limiting on auth endpoints
- Dark mode that persists across sessions
- Integration tests for auth, project, and task flows
- Pagination on all list endpoints

---

## 2. Architecture Decisions

**Backend**

I used `chi` over Gin because it's stdlib-compatible and has no magic. Every handler is a plain `http.HandlerFunc` — easy to trace, easy to test.

No ORM. The schema is small enough that raw `pgx` queries are clearer and cheaper to write. Migrations run automatically on startup via `golang-migrate` — no manual step needed.

Auth is JWT-only with a 24-hour expiry. Bcrypt cost 12 hits a reasonable balance between brute-force resistance and registration latency. The JWT secret is validated at startup to be at least 32 characters — a short secret is as bad as a hardcoded one.

I separated 401 (not authenticated) from 403 (authenticated but not allowed) strictly. Conflating these breaks client-side redirect logic.

Task reordering runs in a transaction: update the moved task, then renumber every task in the affected column so there are no gaps. I collect all IDs from the query before running the updates — mixing an open result set with writes on the same connection causes "conn busy" errors in pgx.

Task deletion checks both `owner_id` on the project and `created_by` on the task, so either the project owner or the person who created the task can delete it.

**Frontend**

React Query manages all server state with a 30s stale time. Zustand holds auth state and is persisted to localStorage so page refreshes don't log you out. Task status changes use optimistic updates — the UI moves immediately and reverts on error.

Radix UI provides accessible dialog, select, and label components. Styling is Tailwind with a custom brand colour. Drag-and-drop uses `@dnd-kit` with `useDroppable` column containers (so empty columns accept drops) and `DragOverlay` for visual feedback during a drag.

**Tradeoffs**

- No refresh tokens — 24h JWT is sufficient for this scope
- No WebSocket real-time sync — React Query polling is fine for individual sessions
- No RBAC beyond owner vs. non-owner — the assignment doesn't require it

---

## 3. Running Locally

You need Docker and Docker Compose v2. Nothing else.

```bash
git clone https://github.com/luvgupta014/taskflow-luvgupta
cd taskflow-luvgupta
cp .env.example .env
docker compose up
```

- Frontend: **http://localhost:3000**
- API: **http://localhost:8080**

First build takes ~60 seconds (Go compile + npm install). After that, subsequent starts reuse cached layers and are fast.

**Clean Start (Fresh Database)**

If you want a completely fresh database (recommended on first run or after experimenting):

```bash
docker compose down -v
docker compose up --build
```

The `-v` flag removes named volumes, so `pgdata` is recreated. Migrations run automatically on startup.

---

## 4. Running Migrations

Migrations run **automatically** when the API container starts. No manual step needed.

To run them manually against a local Postgres instance:

```bash
migrate -path ./backend/migrations \
        -database "postgres://postgres:postgres@localhost:5432/taskflow?sslmode=disable" \
        up
```

---

## 5. Test Credentials

```
Email:    test@example.com
Password: password123
```

The seed also creates a "Website Redesign" project with three tasks in different statuses (todo, in_progress, done) so you can see the UI populated immediately.

---

## 6. API Reference

All endpoints return `Content-Type: application/json`. Protected endpoints require `Authorization: Bearer <token>`.

**Auth**
```
POST /auth/register   { name, email, password }  → 201 { token, user }
POST /auth/login      { email, password }         → 200 { token, user }
```

**Projects** *(all require Bearer token)*
```
GET    /projects?page=1&limit=20   → { projects, page, limit }
POST   /projects                   { name, description? }           → 201 project
GET    /projects/:id               → project + tasks array
PATCH  /projects/:id               { name?, description? }          → project (owner only)
DELETE /projects/:id               → 204 (owner only, cascades tasks)
GET    /projects/:id/stats         → { by_status, by_assignee }
GET    /projects/:id/members       → { members: [{ id, name, email }] }
```

**Tasks** *(all require Bearer token)*
```
GET    /projects/:id/tasks?status=&assignee=&page=1&limit=50
POST   /projects/:id/tasks   { title, status?, priority?, assignee_id?, due_date? }  → 201 task
PATCH  /tasks/:id            partial task fields + optional order  → updated task
DELETE /tasks/:id            → 204 (project owner or task creator)
```

**Error responses**
```json
{ "error": "validation failed", "fields": { "email": "is required" } }  // 400
{ "error": "unauthorized" }                                               // 401
{ "error": "forbidden" }                                                  // 403
{ "error": "not found" }                                                  // 404
{ "error": "too many requests, please try again later" }                  // 429
```

---

## 7. What I'd Do With More Time

- **Refresh tokens.** Short-lived access tokens with a sliding-window refresh would be better for real products, but 24h is fine here.
- **Real-time updates.** SSE or WebSocket so two people working on the same project see each other's changes live. The current stale-time polling works for single-user but would feel laggy in a team.
- **Better test coverage.** The integration tests cover the main flows but not edge cases — concurrent reorders, pagination boundaries, rate limit recovery.
- **Search.** Full-text search on task title and description would be the first usability improvement users ask for.
- **Audit log.** Who changed what and when. Low effort to add (an `events` table + a few inserts) and high value for accountability.

---
