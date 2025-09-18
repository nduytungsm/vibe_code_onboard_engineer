package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

func (hc *HealthController) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"message": "Server is running",
		"service": "repo-explanation",
	})
}
