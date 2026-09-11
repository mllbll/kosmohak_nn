package config

import "os"

type Config struct {
	Addr         string
	PythonBin    string
	RunnerScript string
}

func Load() Config {
	return Config{
		Addr:         getenv("HTTP_ADDR", ":8080"),
		PythonBin:    getenv("PYTHON_BIN", "python3"),
		RunnerScript: getenv("RUNNER_SCRIPT", "python/runner.py"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
