package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"repo-explanation/controllers"
	"repo-explanation/routes"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize controllers
	healthController := controllers.NewHealthController()

	// Setup routes
	routes.SetupRoutes(e, healthController)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
