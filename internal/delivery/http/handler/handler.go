package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/whenipush/envgate/internal/entity"
)

type ProjectService interface {
	GetProject(ctx context.Context, name string) (*entity.Project, error)
	SaveProject(ctx context.Context, name string, environments map[string]*entity.ProjectEnv) error
	ListProjectsWithEnvs(ctx context.Context) (map[string][]string, error)
	UpdateProjectEnv(ctx context.Context, name string, newEnvironments map[string]*entity.ProjectEnv) error
	DeleteProject(ctx context.Context, name string) error
}

type TokenService interface {
	GenerateToken(ctx context.Context, projectName, environment, user string) (string, error)
	RevokeToken(ctx context.Context, tokenStr string) error
	ListTokens(ctx context.Context, projectName string) (map[string]*entity.TokenMeta, error)
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

	proj, err := h.projectSvc.GetProject(c.Request.Context(), projectName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load project")
		return
	}
	if proj == nil {
		c.String(http.StatusNotFound, "Project not found")
		return
	}

	// Получаем список токенов, привязанных к этому проекту
	tokens, err := h.tokenSvc.ListTokens(c.Request.Context(), projectName)
	if err != nil {
		// Если список не загрузился, не ломаем всю страницу, а просто инициализируем пустую мапу
		tokens = make(map[string]*entity.TokenMeta)
	}

	c.HTML(http.StatusOK, "base", gin.H{
		"Page":    "project_detail",
		"Project": proj,
		"Tokens":  tokens,
	})
}

func (h *Handler) CreateProject(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.String(http.StatusBadRequest, "Project name is required")
		return
	}

	envsInput := c.PostForm("environments")

	if envsInput == "" {
		envsInput = "development, production"
	}

	rawEnvs := strings.Split(envsInput, ",")
	envs := make(map[string]*entity.ProjectEnv)

	for _, env := range rawEnvs {
		envName := strings.TrimSpace(env)
		if envName == "" {
			continue
		}

		envs[envName] = &entity.ProjectEnv{
			Variables: make(map[string]string),
		}
	}

	if len(envs) == 0 {
		c.String(http.StatusBadRequest, "At least one valid environment is required")
		return
	}

	err := h.projectSvc.SaveProject(c.Request.Context(), name, envs)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create project: %v", err)
		return
	}

	c.Header("HX-Redirect", "/admin/projects")
	c.Status(http.StatusSeeOther)
}

func (h *Handler) UpdateProjectEnv(c *gin.Context) {
	projectName := c.Param("name")
	environment := c.PostForm("environment")

	if environment == "" {
		c.String(http.StatusBadRequest, "Environment name is required")
		return
	}

	keys := c.PostFormArray("key[]")
	values := c.PostFormArray("value[]")

	proj, err := h.projectSvc.GetProject(c.Request.Context(), projectName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load project: %v", err)
		return
	}
	if proj == nil {
		c.String(http.StatusNotFound, "Project not found")
		return
	}

	variables := make(map[string]string)
	for i, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" && i < len(values) {
			variables[key] = values[i]
		}
	}

	if proj.Environments == nil {
		proj.Environments = make(map[string]*entity.ProjectEnv)
	}

	proj.Environments[environment] = &entity.ProjectEnv{
		Variables: variables,
	}

	err = h.projectSvc.UpdateProjectEnv(c.Request.Context(), projectName, proj.Environments)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to save project: %v", err)
		return
	}

	c.String(http.StatusOK, "<div class='text-emerald-400 font-medium text-sm'>Сохранено успешно!</div>")
}

func (h *Handler) DeleteProject(c *gin.Context) {
	projectName := c.Param("name")

	if err := h.projectSvc.DeleteProject(c.Request.Context(), projectName); err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete project")
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) GenerateToken(c *gin.Context) {
	projectName := c.PostForm("project_name")
	environment := c.PostForm("environment")
	user := c.PostForm("user")

	if user == "" {
		c.String(http.StatusBadRequest, "Поле 'Для кого / Назначение' обязательно")
		return
	}

	token, err := h.tokenSvc.GenerateToken(c.Request.Context(), projectName, environment, user)
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
func (h *Handler) RevokeToken(c *gin.Context) {
	tokenStr := c.Param("token")
	if tokenStr == "" {
		c.String(http.StatusBadRequest, "Идентификатор токена не указан")
		return
	}

	err := h.tokenSvc.RevokeToken(c.Request.Context(), tokenStr)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не удалось отозвать токен")
		return
	}

	c.Status(http.StatusOK)
}
