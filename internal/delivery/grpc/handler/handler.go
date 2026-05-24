package handler

import (
	"context"

	envgatev1 "github.com/whenipush/envgate/gen/go/envgate/v1"
	"github.com/whenipush/envgate/internal/entity"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TokenService interface {
	GetTokenMeta(ctx context.Context, tokenStr string) (*entity.TokenMeta, error)
}

type ProjectService interface {
	GetProject(ctx context.Context, name []byte) (*entity.Project, error)
}

type EnvGateHandler struct {
	envgatev1.UnimplementedEnvGateServiceServer

	tokenService   TokenService
	projectService ProjectService
}

func NewHandler(ts TokenService, ps ProjectService) *EnvGateHandler {
	return &EnvGateHandler{
		tokenService:   ts,
		projectService: ps,
	}
}

func (h *EnvGateHandler) PullSecrets(ctx context.Context, req *envgatev1.PullSecretsRequest) (*envgatev1.PullSecretsResponse, error) {
	if req.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	meta, err := h.tokenService.GetTokenMeta(ctx, req.GetToken())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	project, err := h.projectService.GetProject(ctx, []byte(meta.ProjectName))
	if err != nil || project == nil {
		return nil, status.Error(codes.NotFound, "associated project not found")
	}

	envData, ok := project.Environments[meta.Environment]
	if !ok || envData == nil || envData.Variables == nil {
		return nil, status.Error(codes.NotFound, "requested environment not configured in project")
	}

	return &envgatev1.PullSecretsResponse{
		ProjectName: meta.ProjectName,
		Environment: meta.Environment,
		Variables:   envData.Variables,
	}, nil
}
