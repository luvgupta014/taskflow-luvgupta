# TaskFlow

A full stack task management system built with Go, React, and PostgreSQL.

---

## 1. Overview

TaskFlow lets teams create projects, break them into tasks, and track progress from todo through done. It is a small but complete product — not a prototype — with JWT authentication, a relational data model, a REST API, and a responsive UI.

**Stack**

| Layer | Technology |
|---|---|
| Backend | Go 1.22, chi router, pgx/v5, golang-migrate |
| Frontend | React 18, TypeScript, Vite, TailwindCSS, Radix UI |
| Database | PostgreSQL 15 |
| Infrastructure | Docker, docker compose |

---

## 2. Architecture Decisions

**Backend**

I chose `chi` over a heavier framework like Gin because it is stdlib-compatible (`net/http`), has zero magic, and its middleware model is the same interface composition that Go is built around. Every handler is a plain function — easy to test, easy to read.

I deliberately avoided an ORM. The schema is simple enough that raw `pgx` queries are faster to write, easier to audit, and produce no surprise SQL. Migrations are managed by `golang-migrate` and run automatically on startup via an embedded `file://` source — no manual migration step.

Auth is JWT-only. Sessions would add statefulness without meaningful security benefit at this scale. Bcrypt cost is 12, which keeps brute-force expensive while keeping registration under 200 ms on commodity hardware.

Error handling draws a hard line between 401 (unauthenticated) and 403 (authenticated but not allowed). These are different conditions and conflating them breaks client logic.

**Frontend**

React Query handles all server state. Zustand holds auth state and is persisted to localStorage, so a page refresh keeps the user logged in. Optimistic updates on task status changes make the UI feel instant — if the server rejects, the previous state is restored from the React Query snapshot.

I used Radix UI primitives for the modal and select components. This gives correct keyboard navigation and ARIA semantics without shipping a full component library. Styling is done with Tailwind and a custom brand colour token.

**What I left out intentionally**

- Role-based permissions beyond owner/non-owner: the schema supports it via `assignee_id`, but a full RBAC layer would take a day and is not in the rubric.
- Refresh tokens: a 24-hour JWT is sufficient for an assignment and keeps the auth surface small.
- WebSocket real-time sync: the HTTP polling model is reliable and far simpler to operate. Adding SSE or WebSocket would require a session store and is treated as a bonus.
- Pagination: endpoints return all records. With a sensible `LIMIT` in production queries, this is a one-afternoon addition that does not change any interfaces.

---

## 3. Running Locally

Requires Docker and docker compose (v2). Nothing else.

```bash
git clone https://github.com/luvgupta014/taskflow-luvgupta
cd taskflow-luvgupta
cp .env.example .env
docker compose up
```

The frontend is available at **http://localhost:3000**.
The API is available at **http://localhost:8080**.

First startup takes a minute while Go compiles and npm installs. Subsequent starts are fast.

---

## 4. Running Migrations

Migrations run automatically when the api container starts. They are embedded in the binary and applied via `golang-migrate` before the HTTP server binds.

If you want to run them manually against a local Postgres instance:

```bash
migrate -path ./backend/migrations \
        -database "postgres://postgres:postgres@localhost:5432/taskflow?sslmode=disable" \
        up
```

---

## 5. Test Credentials

A seed user is created on first startup:

```
Email:    test@example.com
Password: password123
```

The seed also creates one project ("Website Redesign") with three tasks in different statuses so you can see the UI populated.

---

## 6. API Reference

All endpoints return `Content-Type: application/json`. Protected endpoints require `Authorization: Bearer <token>`.

### Auth

```
POST /auth/register
Body: { "name": "string", "email": "string", "password": "string" }
201:  { "token": "string", "user": { "id", "name", "email" } }

POST /auth/login
Body: { "email": "string", "password": "string" }
200:  { "token": "string", "user": { "id", "name", "email" } }
```

### Projects

```
GET    /projects           → { "projects": [...] }
POST   /projects           Body: { "name", "description?" } → 201 project
GET    /projects/:id       → project + tasks array
PATCH  /projects/:id       Body: { "name?", "description?" } → updated project
DELETE /projects/:id       → 204
GET    /projects/:id/stats → { "by_status": {...}, "by_assignee": {...} }
```

### Tasks

```
GET    /projects/:id/tasks?status=&assignee=  → { "tasks": [...] }
POST   /projects/:id/tasks                    Body: task fields → 201 task
PATCH  /tasks/:id                             Body: partial task → updated task
DELETE /tasks/:id                             → 204
```

### Error shape

```json
{ "error": "validation failed", "fields": { "email": "is required" } }
{ "error": "unauthorized" }
{ "error": "forbidden" }
{ "error": "not found" }
```

---

## 7. What I'd Do With More Time

**Testing.** I have zero automated tests here, which I am not proud of. The first thing I would add is a table-driven integration test suite hitting a test database — auth flows, project ownership checks, and task filter correctness. These are the highest-value tests for an API this size.

**Pagination.** Every list endpoint returns all records. Adding `?page=&limit=` with a `Link` header is straightforward but I ran out of runway.

**Drag-and-drop.** The UI has a flat task list. A Kanban board with `@dnd-kit` columns per status would be a better task management experience.

**Input sanitisation.** The current validation is basic (required, pattern). A production service would need length caps, rate limiting on auth endpoints, and stricter content-type enforcement.

**Config management.** All config comes from env vars, which is correct, but there is no validation that JWT_SECRET meets a minimum entropy requirement. A weak secret is as bad as a hardcoded one.
