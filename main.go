package main

import (
	"flag"
	"fmt"
	"os"

	"repo-explanation/cli"
	"repo-explanation/controllers"
	"repo-explanation/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	mode := flag.String("mode", "server", "Mode to run: 'server' or 'cli'")
	flag.Parse()

	switch *mode {
	case "server":
		runServer()
	case "cli":
		runCLI()
	default:
		fmt.Printf("Unknown mode: %s\n", *mode)
		fmt.Println("Available modes: server, cli")
		os.Exit(1)
	}
}

func runServer() {
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

func runCLI() {
	repl := cli.NewREPL()
	repl.Start()
}
