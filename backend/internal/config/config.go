package config

import (
	"os"
	"strconv"
	"strings"

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
			// Banana ripeness classes; must match the model's training order.
			Labels:        getenvList("MODEL_LABELS", []string{"unripe", "ripe", "overripe"}),
			ConfThreshold: getenvFloat("MODEL_CONF_THRESHOLD", 0.25),
			IoUThreshold:  getenvFloat("MODEL_IOU_THRESHOLD", 0.45),
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

func getenvFloat(key string, def float32) float32 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			return float32(f)
		}
	}
	return def
}

func getenvList(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}
