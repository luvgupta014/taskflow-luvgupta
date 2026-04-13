package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luvgupta014/taskflow/internal/middleware"
	"github.com/luvgupta014/taskflow/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func createTestUser(t *testing.T, pool *pgxpool.Pool, name, email, password string) (uuid.UUID, string) {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	var userID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, name, email, string(hash)).Scan(&userID)

	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return userID, password
}

func createTestProject(t *testing.T, pool *pgxpool.Pool, ownerID uuid.UUID, name, description string) uuid.UUID {
	ctx := context.Background()

	var projectID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO projects (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, name, description, ownerID).Scan(&projectID)

	if err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}

	return projectID
}

func createTestTask(t *testing.T, pool *pgxpool.Pool, projectID uuid.UUID, title, status string) uuid.UUID {
	ctx := context.Background()

	var taskID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO tasks (title, status, project_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, title, status, projectID).Scan(&taskID)

	if err != nil {
		t.Fatalf("failed to create test task: %v", err)
	}

	return taskID
}

func generateToken(userID uuid.UUID, jwtSecret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID.String(),
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))
	return tokenStr
}

func TestProjectOwnershipCheck(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	jwtSecret := "test-secret-key-at-least-32-chars-here"
	projectH := NewProjectHandler(pool)

	// Create two users
	owner := uuid.New()
	otherUser := uuid.New()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, password) VALUES ($1, $2, $3, $4)`, owner, "Owner", "owner@test.com", "hash")
	if err != nil && err.Error() != "no rows in result set" {
		t.Fatalf("failed to create owner user: %v", err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, password) VALUES ($1, $2, $3, $4)`, otherUser, "Other", "other@test.com", "hash")
	if err != nil && err.Error() != "no rows in result set" {
		t.Fatalf("failed to create other user: %v", err)
	}

	// Create a project as owner
	projectID := createTestProject(t, pool, owner, "Test Project", "A test project")

	tests := []struct {
		name           string
		userID         uuid.UUID
		expectedStatus int
		wantErr        bool
	}{
		{
			name:           "owner can view project",
			userID:         owner,
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "non-owner cannot view project",
			userID:         otherUser,
			expectedStatus: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := generateToken(tt.userID, jwtSecret)

			req := httptest.NewRequest("GET", fmt.Sprintf("/projects/%s", projectID.String()), nil)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			// Add context with user ID extracted from token
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, tt.userID.String())
			req = req.WithContext(ctx)

			// Set URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", projectID.String())
			req = req.WithContext(chi.RouteContext(req.Context(), rctx))

			w := httptest.NewRecorder()
			projectH.Get(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestProjectCRUD(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	jwtSecret := "test-secret-key-at-least-32-chars-here"
	projectH := NewProjectHandler(pool)

	userID := uuid.New()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, password) VALUES ($1, $2, $3, $4)`, userID, "Test", "test@test.com", "hash")
	if err != nil && err.Error() != "no rows in result set" {
		t.Fatalf("failed to create user: %v", err)
	}

	token := generateToken(userID, jwtSecret)

	// Test CREATE
	t.Run("create project", func(t *testing.T) {
		body := map[string]string{
			"name":        "New Project",
			"description": "A new project",
		}
		payload, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(payload))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID.String())
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		projectH.Create(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var project model.Project
		json.Unmarshal(w.Body.Bytes(), &project)

		// Test UPDATE
		t.Run("update project", func(t *testing.T) {
			updateBody := map[string]string{
				"name": "Updated Project",
			}
			updatePayload, _ := json.Marshal(updateBody)

			updateReq := httptest.NewRequest("PATCH", fmt.Sprintf("/projects/%s", project.ID), bytes.NewReader(updatePayload))
			updateReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			ctx := context.WithValue(updateReq.Context(), middleware.UserIDKey, userID.String())
			updateReq = updateReq.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", project.ID.String())
			updateReq = updateReq.WithContext(chi.RouteContext(updateReq.Context(), rctx))

			updateW := httptest.NewRecorder()
			projectH.Update(updateW, updateReq)

			if updateW.Code != http.StatusOK {
				t.Errorf("expected %d, got %d", http.StatusOK, updateW.Code)
			}
		})

		// Test DELETE
		t.Run("delete project", func(t *testing.T) {
			deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/projects/%s", project.ID), nil)
			deleteReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			ctx := context.WithValue(deleteReq.Context(), middleware.UserIDKey, userID.String())
			deleteReq = deleteReq.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", project.ID.String())
			deleteReq = deleteReq.WithContext(chi.RouteContext(deleteReq.Context(), rctx))

			deleteW := httptest.NewRecorder()
			projectH.Delete(deleteW, deleteReq)

			if deleteW.Code != http.StatusNoContent {
				t.Errorf("expected %d, got %d", http.StatusNoContent, deleteW.Code)
			}
		})
	})
}
