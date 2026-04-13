package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

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
		       to_char(due_date, 'YYYY-MM-DD'), "order", created_by, created_at, updated_at
		FROM tasks WHERE project_id = $1
	`
	args := []any{projectID}
	idx := 2

	if status := r.URL.Query().Get("status"); status != "" {
		query += ` AND status = $` + strconv.Itoa(idx)
		args = append(args, status)
		idx++
	}
	if assignee := r.URL.Query().Get("assignee"); assignee != "" {
		if aid, err := uuid.Parse(assignee); err == nil {
			query += ` AND assignee_id = $` + strconv.Itoa(idx)
			args = append(args, aid)
			idx++
		}
	}

	query += ` ORDER BY "order" ASC, created_at ASC`

	page, limit := 1, 50
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	offset := (page - 1) * limit
	query += ` LIMIT $` + strconv.Itoa(idx) + ` OFFSET $` + strconv.Itoa(idx+1)
	args = append(args, limit, offset)

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
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.Order, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			slog.Error("tasks: scan", "err", err)
			response.InternalError(w)
			return
		}
		tasks = append(tasks, t)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
		"page":  page,
		"limit": limit,
	})
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
		Title       string             `json:"title"`
		Description *string            `json:"description"`
		Status      model.TaskStatus   `json:"status"`
		Priority    model.TaskPriority `json:"priority"`
		AssigneeID  *uuid.UUID         `json:"assignee_id"`
		DueDate     *string            `json:"due_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	fields := map[string]string{}
	if req.Title == "" {
		fields["title"] = "is required"
	}
	if len(req.Title) > 255 {
		fields["title"] = "must be less than 255 characters"
	}
	if req.Description != nil && len(*req.Description) > 2000 {
		fields["description"] = "must be less than 2000 characters"
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
		INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date,
		                   "order", created_by, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7::date,
		        COALESCE((SELECT MAX("order") FROM tasks WHERE project_id = $5 AND status = $3::varchar), 0) + 1,
		        $8, now(), now())
		RETURNING id, title, description, status, priority, project_id, assignee_id,
		          to_char(due_date, 'YYYY-MM-DD'), "order", created_by, created_at, updated_at
	`, req.Title, req.Description, req.Status, req.Priority, projectID, req.AssigneeID, req.DueDate, userID,
	).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.Order, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid request"})
		return
	}

	var req struct {
		Title       *string             `json:"title"`
		Description *string             `json:"description"`
		Status      *model.TaskStatus   `json:"status"`
		Priority    *model.TaskPriority `json:"priority"`
		AssigneeID  *uuid.UUID          `json:"assignee_id"`
		DueDate     *string             `json:"due_date"`
		Order       *int                `json:"order"`
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	fields := map[string]string{}
	if req.Title != nil && len(*req.Title) > 255 {
		fields["title"] = "must be less than 255 characters"
	}
	if req.Description != nil && len(*req.Description) > 2000 {
		fields["description"] = "must be less than 2000 characters"
	}
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	var rawMap map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawMap)
	_, assigneeExplicit := rawMap["assignee_id"]

	var oldStatus model.TaskStatus
	err = h.db.QueryRow(r.Context(), `SELECT status FROM tasks WHERE id = $1`, taskID).Scan(&oldStatus)
	if err != nil {
		slog.Error("tasks: get current", "err", err)
		response.InternalError(w)
		return
	}

	newStatus := oldStatus
	if req.Status != nil {
		newStatus = *req.Status
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		slog.Error("tasks: begin tx", "err", err)
		response.InternalError(w)
		return
	}
	defer tx.Rollback(r.Context())

	if newStatus != oldStatus && req.Order == nil {
		var maxOrder int
		err = tx.QueryRow(r.Context(),
			`SELECT COALESCE(MAX("order"), -1) FROM tasks WHERE project_id = $1 AND status = $2`,
			projectID, newStatus).Scan(&maxOrder)
		if err != nil {
			slog.Error("tasks: get max order", "err", err)
			response.InternalError(w)
			return
		}
		o := maxOrder + 1
		req.Order = &o
	}

	var t model.Task
	if assigneeExplicit {
		err = tx.QueryRow(r.Context(), `
			UPDATE tasks SET
				title       = COALESCE($1, title),
				description = COALESCE($2, description),
				status      = COALESCE($3, status),
				priority    = COALESCE($4, priority),
				assignee_id = $5,
				due_date    = COALESCE($6::date, due_date),
				"order"     = COALESCE($8, "order"),
				updated_at  = now()
			WHERE id = $7
			RETURNING id, title, description, status, priority, project_id, assignee_id,
			          to_char(due_date, 'YYYY-MM-DD'), "order", created_by, created_at, updated_at
		`, req.Title, req.Description, req.Status, req.Priority, req.AssigneeID, req.DueDate, taskID, req.Order,
		).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.Order, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	} else {
		err = tx.QueryRow(r.Context(), `
			UPDATE tasks SET
				title       = COALESCE($1, title),
				description = COALESCE($2, description),
				status      = COALESCE($3, status),
				priority    = COALESCE($4, priority),
				assignee_id = COALESCE($5, assignee_id),
				due_date    = COALESCE($6::date, due_date),
				"order"     = COALESCE($8, "order"),
				updated_at  = now()
			WHERE id = $7
			RETURNING id, title, description, status, priority, project_id, assignee_id,
			          to_char(due_date, 'YYYY-MM-DD'), "order", created_by, created_at, updated_at
		`, req.Title, req.Description, req.Status, req.Priority, req.AssigneeID, req.DueDate, taskID, req.Order,
		).Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.Order, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	}
	if err != nil {
		slog.Error("tasks: update", "err", err)
		response.InternalError(w)
		return
	}

	reorderRows, err := tx.Query(r.Context(),
		`SELECT id FROM tasks WHERE project_id = $1 AND status = $2 ORDER BY "order" ASC, created_at ASC`,
		projectID, newStatus)
	if err != nil {
		slog.Error("tasks: get tasks for reorder", "err", err)
		response.InternalError(w)
		return
	}

	var newColIDs []uuid.UUID
	for reorderRows.Next() {
		var tid uuid.UUID
		if err := reorderRows.Scan(&tid); err != nil {
			slog.Error("tasks: scan for reorder", "err", err)
			reorderRows.Close()
			response.InternalError(w)
			return
		}
		newColIDs = append(newColIDs, tid)
	}
	reorderRows.Close()

	for orderNum, tid := range newColIDs {
		if _, err = tx.Exec(r.Context(), `UPDATE tasks SET "order" = $1 WHERE id = $2`, orderNum, tid); err != nil {
			slog.Error("tasks: renumber", "err", err)
			response.InternalError(w)
			return
		}
	}

	if newStatus != oldStatus {
		oldRows, err := tx.Query(r.Context(),
			`SELECT id FROM tasks WHERE project_id = $1 AND status = $2 ORDER BY "order" ASC, created_at ASC`,
			projectID, oldStatus)
		if err != nil {
			slog.Error("tasks: get old column tasks", "err", err)
			response.InternalError(w)
			return
		}

		var oldColIDs []uuid.UUID
		for oldRows.Next() {
			var tid uuid.UUID
			if err := oldRows.Scan(&tid); err != nil {
				slog.Error("tasks: scan old column", "err", err)
				oldRows.Close()
				response.InternalError(w)
				return
			}
			oldColIDs = append(oldColIDs, tid)
		}
		oldRows.Close()

		for n, tid := range oldColIDs {
			if _, err = tx.Exec(r.Context(), `UPDATE tasks SET "order" = $1 WHERE id = $2`, n, tid); err != nil {
				slog.Error("tasks: renumber old", "err", err)
				response.InternalError(w)
				return
			}
		}
	}

	if err = tx.Commit(r.Context()); err != nil {
		slog.Error("tasks: commit", "err", err)
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
	var createdBy *uuid.UUID
	err = h.db.QueryRow(r.Context(),
		"SELECT project_id, created_by FROM tasks WHERE id = $1", taskID,
	).Scan(&projectID, &createdBy)
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

	isOwner := ownerID == userID
	isCreator := createdBy != nil && *createdBy == userID

	if !isOwner && !isCreator {
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
