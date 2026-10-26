package routers

import (
	"net/http"
	"strings"

	docs "github.com/emiliogozo/panahon-api-go/internal/docs/public"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func redirectToDocsIndex(c *gin.Context) {
	path := strings.TrimRight(c.Request.URL.Path, "/")
	c.Redirect(http.StatusFound, path+"/index.html")
}

func wrapDocsIndex(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Param("any") {
		case "", "/":
			redirectToDocsIndex(c)
		default:
			handler(c)
		}
	}
}

func registerDocsRoutes(api *gin.RouterGroup) {
	fullDocs := ginSwagger.WrapHandler(swaggerFiles.Handler)
	api.GET("/docs", redirectToDocsIndex)
	api.GET("/docs/*any", wrapDocsIndex(fullDocs))

	publicDocs := ginSwagger.CustomWrapHandler(&ginSwagger.Config{
		URL:          "../public-openapi.json",
		Title:        "Panahon Public API",
		DocExpansion: "list",
		DeepLinking:  true,
	}, swaggerFiles.NewHandler())
	api.GET("/public-docs", redirectToDocsIndex)
	api.GET("/public-docs/*any", wrapDocsIndex(publicDocs))

	api.GET("/public-openapi.json", func(c *gin.Context) {
		spec, err := docs.PublicSpec()
		if err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot build public API docs"})
			return
		}

		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "application/json; charset=utf-8", spec)
	})
}
