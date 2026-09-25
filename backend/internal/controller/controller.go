package controller

// Controllers groups every HTTP controller so the router can be wired in one place.
// Add new controllers here as the API grows.
type Controllers struct {
	Health    *HealthController
	Image     *ImageController
	Inference *InferenceController
}
