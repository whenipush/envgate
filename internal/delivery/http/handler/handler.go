package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/whenipush/envgate/internal/entity"
)

type ProjectService interface {
	GetProject(ctx context.Context, name []byte) (*entity.Project, error)
	SaveProject(ctx context.Context, project *entity.Project) error
	ListProjectsWithEnvs(ctx context.Context) (map[string][]string, error)
	UpdateProjectEnv(ctx context.Context, oldName string, newName *string, newEnvironments map[string]*entity.ProjectEnv) error
	DeleteProject(ctx context.Context, name []byte) error
}

type TokenService interface {
	GenerateToken(ctx context.Context, projectName, environment string) (string, error)
}

type Handler struct {
	projectSvc ProjectService
	tokenSvc   TokenService
}

func NewHandler(projectSvc ProjectService, tokenSvc TokenService) *Handler {
	return &Handler{
		projectSvc: projectSvc,
		tokenSvc:   tokenSvc,
	}
}

func (h *Handler) ListProjects(c *gin.Context) {
	rawProjects, err := h.projectSvc.ListProjectsWithEnvs(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load projects: %v", err)
		return
	}

	c.HTML(http.StatusOK, "base", gin.H{
		"Page":     "projects",
		"Projects": rawProjects,
	})
}

func (h *Handler) GetProject(c *gin.Context) {
	projectName := c.Param("name")

	project, err := h.projectSvc.GetProject(c.Request.Context(), []byte(projectName))
	if err != nil {
		c.String(http.StatusNotFound, "Project not found")
		return
	}

	c.HTML(http.StatusOK, "base", gin.H{
		"Page":    "project_detail",
		"Project": project,
	})
}

func (h *Handler) CreateProject(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.String(http.StatusBadRequest, "Project name is required")
		return
	}

	envs := map[string]*entity.ProjectEnv{
		"development": {Variables: make(map[string]string)},
		"production":  {Variables: make(map[string]string)},
	}

	err := h.projectSvc.SaveProject(c.Request.Context(), &entity.Project{
		Name:         name,
		Environments: envs,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create project: %v", err)
		return
	}

	c.Header("HX-Redirect", "/admin/projects")
	c.Status(http.StatusSeeOther)
}

func (h *Handler) UpdateProject(c *gin.Context) {
	projectName := c.Param("name")

	keys := c.PostFormArray("key[]")
	values := c.PostFormArray("value[]")
	environment := c.PostForm("environment")

	variables := make(map[string]string)
	for i, key := range keys {
		if key != "" && i < len(values) {
			variables[key] = values[i]
		}
	}

	envs := map[string]*entity.ProjectEnv{
		environment: {Variables: variables},
	}

	err := h.projectSvc.UpdateProjectEnv(c.Request.Context(), projectName, &projectName, envs)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to update env: %v", err)
		return
	}

	c.String(http.StatusOK, "<div class='text-emerald-400 font-medium text-sm'>Сохранено успешно!</div>")
}

func (h *Handler) DeleteProject(c *gin.Context) {
	projectName := c.Param("name")

	if err := h.projectSvc.DeleteProject(c.Request.Context(), []byte(projectName)); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete project")
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) GenerateToken(c *gin.Context) {
	projectName := c.PostForm("project_name")
	environment := c.PostForm("environment")

	token, err := h.tokenSvc.GenerateToken(c.Request.Context(), projectName, environment)
	if err != nil {
		c.String(http.StatusInternalServerError, "Token generation failed")
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `
        <div class="mt-2 p-3 bg-slate-900 border border-emerald-500/30 rounded-lg flex items-center justify-between">
            <code class="text-emerald-400 text-xs break-all select-all">`+token+`</code>
            <span onclick="copyToken(this)" class="text-[10px] bg-emerald-500/20 text-emerald-300 px-2 py-1 rounded ml-2 cursor-pointer hover:bg-emerald-500/30 transition-all select-none">Копировать</span>
        </div>
    `)
}
