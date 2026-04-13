package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luvgupta014/taskflow/internal/config"
	"github.com/luvgupta014/taskflow/internal/db"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to create DB pool: %v", err)
	}

	return pool
}

func TestAuthRegister(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	authH := NewAuthHandler(pool, "test-secret-key-at-least-32-chars")

	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
		wantErr        bool
	}{
		{
			name: "valid registration",
			body: map[string]string{
				"name":     "Test User",
				"email":    "test-" + uuid.New().String() + "@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusCreated,
			wantErr:        false,
		},
		{
			name: "missing email",
			body: map[string]string{
				"name":     "Test User",
				"email":    "",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name: "missing password",
			body: map[string]string{
				"name":     "Test User",
				"email":    "test-" + uuid.New().String() + "@example.com",
				"password": "",
			},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
		{
			name: "missing name",
			body: map[string]string{
				"name":     "",
				"email":    "test-" + uuid.New().String() + "@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(body))
			w := httptest.NewRecorder()

			authH.Register(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d", tt.expectedStatus, w.Code)
			}

			if !tt.wantErr && w.Code == http.StatusCreated {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["token"]; !ok {
					t.Error("expected token in response")
				}
				if _, ok := resp["user"]; !ok {
					t.Error("expected user in response")
				}
			}
		})
	}
}

func TestAuthLogin(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	authH := NewAuthHandler(pool, "test-secret-key-at-least-32-chars")

	// First register a user
	email := "test-login-" + uuid.New().String() + "@example.com"
	password := "password123"
	registerBody := map[string]string{
		"name":     "Test User",
		"email":    email,
		"password": password,
	}
	registerPayload, _ := json.Marshal(registerBody)
	registerReq := httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerPayload))
	registerW := httptest.NewRecorder()
	authH.Register(registerW, registerReq)

	if registerW.Code != http.StatusCreated {
		t.Fatalf("registration failed: %d", registerW.Code)
	}

	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
		wantErr        bool
	}{
		{
			name: "valid login",
			body: map[string]string{
				"email":    email,
				"password": password,
			},
			expectedStatus: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "invalid email",
			body: map[string]string{
				"email":    "nonexistent@example.com",
				"password": password,
			},
			expectedStatus: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "invalid password",
			body: map[string]string{
				"email":    email,
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
			w := httptest.NewRecorder()

			authH.Login(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected %d, got %d", tt.expectedStatus, w.Code)
			}

			if !tt.wantErr && w.Code == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				if _, ok := resp["token"]; !ok {
					t.Error("expected token in response")
				}
			}
		})
	}
}
