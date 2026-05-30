package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OK sends a successful response with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// Error sends an error response with code and message.
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, gin.H{
		"code":    code,
		"message": message,
		"data":    nil,
	})
}
