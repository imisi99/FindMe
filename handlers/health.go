package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DetailedHealth godoc
// @Summary Checks the health of the services running
// @Description This gives a detailed health status of the running services
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any "Service Healthy"
// @Failure 503 {object} map[string]any "Service Degraded"
// @Router /health/detailed [get]
func (s *Service) DetailedHealth(ctx *gin.Context) {
	health := gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"checks":    gin.H{},
	}

	overallHealthy := true

	// Check Database
	if err := s.DB.CheckHealth(); err != nil {
		health["checks"].(gin.H)["database"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		overallHealthy = false
	} else {
		health["checks"].(gin.H)["database"] = gin.H{
			"status": "healthy",
		}
	}

	// Check Redis
	if err := s.RDB.CheckHealth(); err != nil {
		health["checks"].(gin.H)["cache"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		overallHealthy = false
	} else {
		health["checks"].(gin.H)["cache"] = gin.H{
			"status": "healthy",
		}
	}

	if !overallHealthy {
		health["status"] = "degraded"
		ctx.JSON(http.StatusServiceUnavailable, health)
		return
	}

	ctx.JSON(http.StatusOK, health)
}
