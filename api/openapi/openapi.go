// Package openapi serves the machine-readable API contract shared by the web app and the
// Flutter apps. The document is embedded, so each build serves the spec of its own code.
package openapi

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var Spec []byte

func RegisterRoutes(v1 *gin.RouterGroup) {
	v1.GET("/openapi.yaml", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", Spec)
	})
}
