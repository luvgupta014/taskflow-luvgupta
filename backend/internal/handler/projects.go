package handler

import (
	"encoding/json"
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

type ProjectHandler struct {
	db *pgxpool.Pool
}

func NewProjectHandler(db *pgxpool.Pool) *ProjectHandler {
	return &ProjectHandler{db: db}
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	page, limit := 1, 20
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

	rows, err := h.db.Query(r.Context(), `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		slog.Error("projects: list", "err", err)
		response.InternalError(w)
		return
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt); err != nil {
			slog.Error("projects: scan", "err", err)
			response.InternalError(w)
			return
		}
		projects = append(projects, p)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"projects": projects,
		"page":     page,
		"limit":    limit,
	})
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())

	var req struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}
	if req.Name == "" {
		response.ValidationError(w, map[string]string{"name": "is required"})
		return
	}
	if len(req.Name) > 255 {
		response.ValidationError(w, map[string]string{"name": "must be less than 255 characters"})
		return
	}
	if req.Description != nil && len(*req.Description) > 2000 {
		response.ValidationError(w, map[string]string{"description": "must be less than 2000 characters"})
		return
	}

	var p model.Project
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO projects (id, name, description, owner_id, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, now())
		RETURNING id, name, description, owner_id, created_at
	`, req.Name, req.Description, userID).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
	if err != nil {
		slog.Error("projects: create", "err", err)
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusCreated, p)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	var p model.Project
	err = h.db.QueryRow(r.Context(), `
		SELECT id, name, description, owner_id, created_at FROM projects WHERE id = $1
	`, projectID).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		response.NotFound(w)
		return
	}
	if err != nil {
		slog.Error("projects: get", "err", err)
		response.InternalError(w)
		return
	}

	hasAccess, err := h.userHasAccess(r, projectID, userID)
	if err != nil {
		response.InternalError(w)
		return
	}
	if !hasAccess {
		response.NotFound(w)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT id, title, description, status, priority, project_id, assignee_id,
		       to_char(due_date, 'YYYY-MM-DD'), "order", created_by, created_at, updated_at
		FROM tasks WHERE project_id = $1 ORDER BY status, "order" ASC, created_at ASC
	`, projectID)
	if err != nil {
		slog.Error("projects: get tasks", "err", err)
		response.InternalError(w)
		return
	}
	defer rows.Close()

	p.Tasks = []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.Order, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			slog.Error("projects: scan task", "err", err)
			response.InternalError(w)
			return
		}
		p.Tasks = append(p.Tasks, t)
	}

	response.JSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	var ownerID uuid.UUID
	err = h.db.QueryRow(r.Context(), "SELECT owner_id FROM projects WHERE id = $1", projectID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	if ownerID != userID {
		response.Forbidden(w)
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	fields := map[string]string{}
	if req.Name != nil && len(*req.Name) > 255 {
		fields["name"] = "must be less than 255 characters"
	}
	if req.Description != nil && len(*req.Description) > 2000 {
		fields["description"] = "must be less than 2000 characters"
	}
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	var p model.Project
	err = h.db.QueryRow(r.Context(), `
		UPDATE projects SET
			name = COALESCE($1, name),
			description = COALESCE($2, description)
		WHERE id = $3
		RETURNING id, name, description, owner_id, created_at
	`, req.Name, req.Description, projectID).Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt)
	if err != nil {
		slog.Error("projects: update", "err", err)
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusOK, p)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	var ownerID uuid.UUID
	err = h.db.QueryRow(r.Context(), "SELECT owner_id FROM projects WHERE id = $1", projectID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		response.NotFound(w)
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	if ownerID != userID {
		response.Forbidden(w)
		return
	}

	_, err = h.db.Exec(r.Context(), "DELETE FROM projects WHERE id = $1", projectID)
	if err != nil {
		slog.Error("projects: delete", "err", err)
		response.InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProjectHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	hasAccess, err := h.userHasAccess(r, projectID, userID)
	if err != nil {
		response.InternalError(w)
		return
	}
	if !hasAccess {
		response.NotFound(w)
		return
	}

	statusRows, err := h.db.Query(r.Context(), `
		SELECT status, COUNT(*) FROM tasks WHERE project_id = $1 GROUP BY status
	`, projectID)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer statusRows.Close()

	byStatus := map[string]int{"todo": 0, "in_progress": 0, "done": 0}
	for statusRows.Next() {
		var status string
		var count int
		statusRows.Scan(&status, &count)
		byStatus[status] = count
	}

	assigneeRows, err := h.db.Query(r.Context(), `
		SELECT u.id, u.name, COUNT(t.id)
		FROM tasks t
		JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id = $1 AND t.assignee_id IS NOT NULL
		GROUP BY u.id, u.name
	`, projectID)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer assigneeRows.Close()

	byAssignee := map[string]model.AssigneeStat{}
	for assigneeRows.Next() {
		var id uuid.UUID
		var stat model.AssigneeStat
		assigneeRows.Scan(&id, &stat.Name, &stat.Count)
		byAssignee[id.String()] = stat
	}

	response.JSON(w, http.StatusOK, model.ProjectStats{
		ByStatus:   byStatus,
		ByAssignee: byAssignee,
	})
}

func (h *ProjectHandler) userHasAccess(r *http.Request, projectID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := h.db.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM projects WHERE id = $1 AND (
				owner_id = $2 OR EXISTS(SELECT 1 FROM tasks WHERE project_id = $1 AND assignee_id = $2)
			)
		)
	`, projectID, userID).Scan(&exists)
	return exists, err
}

func (h *ProjectHandler) Members(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserIDFromContext(r.Context())
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.NotFound(w)
		return
	}

	hasAccess, err := h.userHasAccess(r, projectID, userID)
	if err != nil {
		response.InternalError(w)
		return
	}
	if !hasAccess {
		response.NotFound(w)
		return
	}

	rows, err := h.db.Query(r.Context(), `SELECT id, name, email FROM users ORDER BY name`)
	if err != nil {
		slog.Error("projects: members", "err", err)
		response.InternalError(w)
		return
	}
	defer rows.Close()

	type member struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	}
	users := []member{}
	for rows.Next() {
		var u member
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			slog.Error("projects: scan member", "err", err)
			response.InternalError(w)
			return
		}
		users = append(users, u)
	}

	response.JSON(w, http.StatusOK, map[string]any{"members": users})
}
