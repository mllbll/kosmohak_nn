package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr         string
	PythonBin    string
	RunnerScript string
	CORSOrigins  []string
}

func Load() Config {
	return Config{
		Addr:         getenv("HTTP_ADDR", ":8080"),
		PythonBin:    getenv("PYTHON_BIN", "python3"),
		RunnerScript: getenv("RUNNER_SCRIPT", "python/runner.py"),
		CORSOrigins:  splitCSV(getenv("CORS_ORIGINS", "*")),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
