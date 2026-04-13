package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luvgupta014/taskflow/internal/response"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db        *pgxpool.Pool
	jwtSecret string
}

func NewAuthHandler(db *pgxpool.Pool, jwtSecret string) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  userPayload `json:"user"`
}

type userPayload struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	fields := map[string]string{}
	if req.Name == "" {
		fields["name"] = "is required"
	}
	if len(req.Name) > 255 {
		fields["name"] = "must be less than 255 characters"
	}
	if req.Email == "" {
		fields["email"] = "is required"
	}
	if len(req.Email) > 255 {
		fields["email"] = "must be less than 255 characters"
	}
	if len(req.Password) < 8 {
		fields["password"] = "must be at least 8 characters"
	}
	if len(req.Password) > 255 {
		fields["password"] = "must be less than 255 characters"
	}
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	var exists bool
	err := h.db.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		slog.Error("register: check email", "err", err)
		response.InternalError(w)
		return
	}
	if exists {
		response.ValidationError(w, map[string]string{"email": "already in use"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		slog.Error("register: bcrypt", "err", err)
		response.InternalError(w)
		return
	}

	var id uuid.UUID
	var createdAt time.Time
	err = h.db.QueryRow(r.Context(),
		`INSERT INTO users (id, name, email, password, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, now())
		 RETURNING id, created_at`,
		req.Name, req.Email, string(hash),
	).Scan(&id, &createdAt)
	if err != nil {
		slog.Error("register: insert", "err", err)
		response.InternalError(w)
		return
	}

	token, err := h.generateToken(id, req.Email)
	if err != nil {
		slog.Error("register: token", "err", err)
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusCreated, authResponse{
		Token: token,
		User:  userPayload{ID: id, Name: req.Name, Email: req.Email},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.ValidationError(w, map[string]string{"body": "invalid JSON"})
		return
	}

	fields := map[string]string{}
	if req.Email == "" {
		fields["email"] = "is required"
	}
	if req.Password == "" {
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		response.ValidationError(w, fields)
		return
	}

	var id uuid.UUID
	var name, hash string
	err := h.db.QueryRow(r.Context(),
		"SELECT id, name, password FROM users WHERE email = $1", req.Email,
	).Scan(&id, &name, &hash)
	if err != nil {
		response.Unauthorized(w)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		response.Unauthorized(w)
		return
	}

	token, err := h.generateToken(id, req.Email)
	if err != nil {
		slog.Error("login: token", "err", err)
		response.InternalError(w)
		return
	}

	response.JSON(w, http.StatusOK, authResponse{
		Token: token,
		User:  userPayload{ID: id, Name: name, Email: req.Email},
	})
}

func (h *AuthHandler) generateToken(userID uuid.UUID, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
