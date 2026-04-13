package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	ServerPort  string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		dbname := os.Getenv("POSTGRES_DB")
		if user == "" || password == "" || host == "" || dbname == "" {
			return nil, fmt.Errorf("missing required database environment variables")
		}
		if port == "" {
			port = "5432"
		}
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters for sufficient entropy")
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("SERVER_PORT must be a number")
	}

	return &Config{
		DatabaseURL: dbURL,
		JWTSecret:   secret,
		ServerPort:  port,
	}, nil
}
