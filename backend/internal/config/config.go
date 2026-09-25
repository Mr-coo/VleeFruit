package config

import (
	"os"
	"strconv"

	"github.com/Mr-coo/VleeFruit/backend/internal/inference"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port      string
	Inference inference.Config
}

// Load reads configuration from the environment, applying sensible defaults.
func Load() Config {
	return Config{
		Port: getenv("BACKEND_PORT", "8080"),
		Inference: inference.Config{
			ModelPath:     os.Getenv("MODEL_PATH"),
			SharedLibPath: os.Getenv("ONNXRUNTIME_LIB_PATH"),
			InputName:     getenv("MODEL_INPUT_NAME", "images"),
			OutputName:    getenv("MODEL_OUTPUT_NAME", "output0"),
			InputSize:     getenvInt("MODEL_INPUT_SIZE", 640),
		},
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
