package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application and database configuration.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	Name string
	Port int
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// Load reads configuration from environment variables and returns a Config instance.
func Load() (*Config, error) {
	appPort, err := getEnvAsPort("APP_PORT", 3000)
	if err != nil {
		return nil, err
	}

	dbPort, err := getEnvAsPort("DB_PORT", 5432)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "CampusSpace"),
			Port: appPort,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "campus_space"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsPort(key string, defaultValue int) (int, error) {
	rawValue, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s: %w", key, err)
	}

	if value <= 0 || value > 65535 {
		return 0, fmt.Errorf("invalid port range for %s (must be 1-65535): %d", key, value)
	}

	return value, nil
}

