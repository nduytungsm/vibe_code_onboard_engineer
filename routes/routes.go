package routes

import (
	"os"
	
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"repo-explanation/controllers"
)

func SetupRoutes(e *echo.Echo, healthController *controllers.HealthController, analysisController *controllers.AnalysisController) {
	// Health check route
	e.GET("/health", healthController.HealthCheck)
	
	// API routes
	api := e.Group("/api")
	
	// Repository analysis endpoints
	api.POST("/analyze", analysisController.AnalyzeRepository)
	api.POST("/analyze/stream", analysisController.StreamAnalyzeRepository)
	
	// Serve static files if they exist (for combined deployment)
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:   "static",
			Index:  "index.html",
			HTML5:  true,
			Browse: false,
		}))
	}
}
