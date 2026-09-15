package routes

import (
	"session-csrf/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterCsrfRoutes(r *gin.Engine) {
	r.GET("/csrf/token", middleware.IssueCsrfToken)
}
