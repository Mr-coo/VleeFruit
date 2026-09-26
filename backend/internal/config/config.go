package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mr-coo/VleeFruit/backend/internal/inference"
	"github.com/Mr-coo/VleeFruit/backend/internal/llm"
	"github.com/Mr-coo/VleeFruit/backend/internal/mqtt"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port      string
	Inference inference.Config
	LLM       llm.Config
	MQTT      mqtt.Config
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
		LLM: llm.Config{
			APIKey:  os.Getenv("GEMINI_API_KEY"),
			Model:   getenv("GEMINI_MODEL", "gemini-2.0-flash"),
			Prompt:  os.Getenv("GEMINI_PROMPT"),
			Timeout: time.Duration(getenvInt("GEMINI_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		MQTT: mqtt.Config{
			BrokerURL:          os.Getenv("MQTT_BROKER_URL"),
			ClientID:           getenv("MQTT_CLIENT_ID", "vleefruit-backend"),
			Username:           os.Getenv("MQTT_USERNAME"),
			Password:           os.Getenv("MQTT_PASSWORD"),
			RequestTopic:       getenv("MQTT_REQUEST_TOPIC", "vleefruit/request"),
			ResponseTopic:      getenv("MQTT_RESPONSE_TOPIC", "vleefruit/response"),
			ImageRequestTopic:  getenv("MQTT_IMAGE_REQUEST_TOPIC", "vleefruit/image/request"),
			ImageResponseTopic: getenv("MQTT_IMAGE_RESPONSE_TOPIC", "vleefruit/image/response"),
			QoS:                byte(getenvInt("MQTT_QOS", 0)),
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
