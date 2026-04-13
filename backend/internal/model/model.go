package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	Tasks       []Task    `json:"tasks,omitempty"`
}

type TaskStatus string
type TaskPriority string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"

	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description *string      `json:"description,omitempty"`
	Status      TaskStatus   `json:"status"`
	Priority    TaskPriority `json:"priority"`
	ProjectID   uuid.UUID    `json:"project_id"`
	AssigneeID  *uuid.UUID   `json:"assignee_id"`
	DueDate     *string      `json:"due_date,omitempty"`
	Order       int          `json:"order"`
	CreatedBy   *uuid.UUID   `json:"created_by,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type ProjectStats struct {
	ByStatus   map[string]int          `json:"by_status"`
	ByAssignee map[string]AssigneeStat `json:"by_assignee"`
}

type AssigneeStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
