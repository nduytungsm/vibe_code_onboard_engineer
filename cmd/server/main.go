package main

import (
	"fmt"

	"repo-explanation/controllers"
	"repo-explanation/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize controllers
	healthController := controllers.NewHealthController()

	fmt.Println("running into this")

	// Setup routes
	routes.SetupRoutes(e, healthController, nil)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
