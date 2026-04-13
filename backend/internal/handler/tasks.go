package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luvgupta014/taskflow/internal/middleware"
	"github.com/luvgupta014/taskflow/internal/model"
	"github.com/luvgupta014/taskflow/internal/response"
)

type TaskHandler struct {
	db *pgxpool.Pool
}

func NewTaskHandler(db *pgxpool.Pool) *TaskHandler {
	return &TaskHandler{db: db}
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	if !h.canViewProject(r, projectID, userID) {
		response.NotFound(w)
		return
	}

	query := `
		SELECT id, title, description, status, priority, project_id, assignee_id,
		       to_char(due_date, 'YYYY-MM-DD'), created_at, updated_at
		FROM tasks WHERE project_id = $1
	`
	args := []any{projectID}
	idx := 2

	if status := r.URL.Query().Get("status"); status != "" {
		query += ` AND status = $` + itoa(idx)
		args = append(args, status)
		idx++
	}
	if assignee := r.URL.Query().Get("assignee"); assignee != "" {
		if aid, err := uuid.Parse(assignee); err == nil {
			query += ` AND assignee_id = $` + itoa(idx)
			args = append(args, aid)
			idx++
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		slog.Error("tasks: list", "err", err)
		response.InternalError(w)
		return
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			slog.Error("tasks: scan", "err", err)
			response.InternalError(w)
			return
		}
		tasks = append(tasks, t)
	}

	response.JSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	if !h.canViewProject(r, projectID, userID) {
		response.NotFound(w)
		return
	}

	var req struct {
		Title       string       `json:"title"`
		Description *string      `json:"description"`
		Status      model.TaskStatus   `json:"status"`
		Priority    model.TaskPriority `json:"priority"`
		AssigneeID  *uuid.UUID   `json:"assignee_id"`
		DueDate     *string      `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	fields := map[string]string{}
	if req.Title == "" {
		fields["title"] = "is required"
	}
	if req.Status == "" {
		req.Status = model.StatusTodo
	}
	if req.Priority == "" {
		req.Priority = model.PriorityMedium
	}
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	var t model.Task
	err = h.db.QueryRow(r.Context(), `
		INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7::date, now(), now())
		RETURNING id, title, description, status, priority, project_id, assignee_id,
		          to_char(due_date, 'YYYY-MM-DD'), created_at, updated_at
	`, req.Title, req.Description, req.Status, req.Priority, projectID, req.AssigneeID, req.DueDate,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		slog.Error("tasks: create", "err", err)
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusCreated, t)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	var projectID uuid.UUID
	err = h.db.QueryRow(r.Context(), "SELECT project_id FROM tasks WHERE id = $1", taskID).Scan(&projectID)
	if err == pgx.ErrNoRows {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	if !h.canViewProject(r, projectID, userID) {
		response.Forbidden(w)
		return
	}

	var req struct {
		Title       *string            `json:"title"`
		Description *string            `json:"description"`
		Status      *model.TaskStatus  `json:"status"`
		Priority    *model.TaskPriority `json:"priority"`
		AssigneeID  *uuid.UUID         `json:"assignee_id"`
		DueDate     *string            `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	var t model.Task
	err = h.db.QueryRow(r.Context(), `
		UPDATE tasks SET
			title       = COALESCE($1, title),
			description = COALESCE($2, description),
			status      = COALESCE($3, status),
			priority    = COALESCE($4, priority),
			assignee_id = COALESCE($5, assignee_id),
			due_date    = COALESCE($6::date, due_date),
			updated_at  = now()
		WHERE id = $7
		RETURNING id, title, description, status, priority, project_id, assignee_id,
		          to_char(due_date, 'YYYY-MM-DD'), created_at, updated_at
	`, req.Title, req.Description, req.Status, req.Priority, req.AssigneeID, req.DueDate, taskID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		slog.Error("tasks: update", "err", err)
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusOK, t)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	var projectID uuid.UUID
	err = h.db.QueryRow(r.Context(), "SELECT project_id FROM tasks WHERE id = $1", taskID).Scan(&projectID)
	if err == pgx.ErrNoRows {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	var ownerID uuid.UUID
	h.db.QueryRow(r.Context(), "SELECT owner_id FROM projects WHERE id = $1", projectID).Scan(&ownerID)

	if ownerID != userID {
		response.Forbidden(w)
		return
	}

	_, err = h.db.Exec(r.Context(), "DELETE FROM tasks WHERE id = $1", taskID)
	if err != nil {
		slog.Error("tasks: delete", "err", err)
		response.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) canViewProject(r *http.Request, projectID, userID uuid.UUID) bool {
	var exists bool
	h.db.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM projects WHERE id = $1 AND (
				owner_id = $2 OR EXISTS(SELECT 1 FROM tasks WHERE project_id = $1 AND assignee_id = $2)
			)
		)
	`, projectID, userID).Scan(&exists)
	return exists
}

func itoa(n int) string {
	return string(rune('0' + n))
}
