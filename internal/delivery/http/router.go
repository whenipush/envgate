package http

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/whenipush/envgate/internal/delivery/http/handler"
	"github.com/whenipush/envgate/internal/pkg/config"
	"github.com/whenipush/envgate/internal/web"
)

func NewRouter(projectSvc handler.ProjectService, tokenSvc handler.TokenService, cfg *config.ConfigServer) *gin.Engine {
	r := gin.Default()
	tmpl, err := template.ParseFS(web.Files, "templates/*.html")
	if err != nil {
		panic("failed to parse HTML templates: " + err.Error())
	}
	r.SetHTMLTemplate(tmpl)

	h := handler.NewHandler(projectSvc, tokenSvc)

	assetsFS, err := fs.Sub(web.Files, "assets")
	if err != nil {
		panic("failed to align assets folder: " + err.Error())
	}
	r.StaticFS("/assets", http.FS(assetsFS))

	admin := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		cfg.Auth.Username: cfg.Auth.Password,
	}))
	{
		admin.GET("/projects", h.ListProjects)
		admin.GET("/projects/:name", h.GetProject)
		admin.POST("/projects", h.CreateProject)
		admin.POST("/projects/:name", h.UpdateProjectEnv)
		admin.DELETE("/projects/:name", h.DeleteProject)

		admin.POST("/tokens", h.GenerateToken)
		admin.DELETE("/tokens/:token", h.RevokeToken)
	}

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/admin/projects")
	})

	return r
}
