package routes

import (
	"github.com/labstack/echo/v4"
	"repo-explanation/controllers"
)

func SetupRoutes(e *echo.Echo, healthController *controllers.HealthController) {
	// Health check route
	e.GET("/health", healthController.HealthCheck)
}
