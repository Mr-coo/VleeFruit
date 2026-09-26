package main

import (
	"log"

	_ "github.com/Mr-coo/VleeFruit/backend/docs"
	"github.com/Mr-coo/VleeFruit/backend/internal/config"
	"github.com/Mr-coo/VleeFruit/backend/internal/controller"
	"github.com/Mr-coo/VleeFruit/backend/internal/inference"
	"github.com/Mr-coo/VleeFruit/backend/internal/llm"
	"github.com/Mr-coo/VleeFruit/backend/internal/mqtt"
	"github.com/Mr-coo/VleeFruit/backend/internal/router"
	"github.com/joho/godotenv"
)

// @title           VleeFruit API
// @version         1.0
// @description     Backend API for the VleeFruit fruit image analysis service.
// @BasePath        /
func main() {
	// Load .env if present; ignore the error so real env vars still work in Docker.
	_ = godotenv.Load()

	cfg := config.Load()

	// AI model layer. In the default build this is a stub; with `-tags onnx`
	// it loads the real ONNX model (if MODEL_PATH is set).
	runner, err := inference.NewRunner(cfg.Inference)
	if err != nil {
		log.Fatalf("init inference runner: %v", err)
	}
	detector := inference.NewDetector(runner, cfg.Inference)
	defer detector.Close()

	// Vision LLM layer (Gemini). Disabled when GEMINI_API_KEY is unset.
	llmClient := llm.NewClient(cfg.LLM)

	// MQTT layer, running alongside the HTTP server. Disabled when
	// MQTT_BROKER_URL is unset.
	mqttClient := mqtt.NewClient(cfg.MQTT)
	if err := mqttClient.Start(); err != nil {
		log.Fatalf("start mqtt: %v", err)
	}
	defer mqttClient.Stop()

	// Controllers (HTTP handlers). Add more here and wire them in the router.
	controllers := &controller.Controllers{
		Health:    controller.NewHealthController(),
		Image:     controller.NewImageController(llmClient),
		Inference: controller.NewInferenceController(detector),
	}

	r := router.New(controllers)

	log.Printf("server listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
