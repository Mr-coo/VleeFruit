package main

import (
	"log"
	"os"

	"github.com/Mr-coo/VleeFruit/backend/internal/controller"
	"github.com/Mr-coo/VleeFruit/backend/internal/router"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present; ignore the error so real env vars still work in Docker.
	_ = godotenv.Load()

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}

	// Controllers (HTTP handlers). Add more here and wire them in the router.
	controllers := &controller.Controllers{
		Health: controller.NewHealthController(),
	}

	r := router.New(controllers)

	log.Printf("server listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
