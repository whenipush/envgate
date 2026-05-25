package project

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/whenipush/envgate/internal/entity"
	"github.com/whenipush/envgate/internal/pkg/crypto"
)

type Repository interface {
	Get(ctx context.Context, bucket entity.Bucket, key []byte) ([]byte, error)
	Put(ctx context.Context, bucket entity.Bucket, key []byte, value []byte) error
	Delete(ctx context.Context, bucket entity.Bucket, key []byte) error
	ListKeys(ctx context.Context, bucket entity.Bucket) ([][]byte, error)
}

type service struct {
	repo      Repository
	masterKey []byte
}

func NewService(repo Repository, masterKey []byte) *service {
	return &service{
		repo:      repo,
		masterKey: masterKey,
	}
}

func (s *service) GetProject(ctx context.Context, name string) (*entity.Project, error) {
	encryptedData, err := s.repo.Get(ctx, entity.BucketProjects, []byte(name))
	if err != nil {
		return nil, fmt.Errorf("failed to get project from repo: %w", err)
	}

	if encryptedData == nil {
		return nil, fmt.Errorf("project not found")
	}

	decryptedData, err := crypto.Decrypt(encryptedData, s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt project data: %w", err)
	}

	var project entity.Project

	if err := json.Unmarshal(decryptedData, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project JSON: %w", err)
	}

	return &project, nil
}

func (s *service) SaveProject(ctx context.Context, name string, environments map[string]*entity.ProjectEnv) error {

	existingData, err := s.repo.Get(ctx, entity.BucketProjects, []byte(name))
	if err != nil {
		return fmt.Errorf("failed to check existing project: %w", err)
	}

	if existingData != nil {
		return fmt.Errorf("project with name '%s' already exists", name)
	}

	project := entity.Project{
		Name:         name,
		Environments: environments,
	}

	plaintext, err := json.Marshal(project)
	if err != nil {
		return err
	}

	encryptedData, err := crypto.Encrypt(plaintext, s.masterKey)
	if err != nil {
		return err
	}

	return s.repo.Put(ctx, entity.BucketProjects, []byte(name), encryptedData)
}

func (s *service) UpdateProjectEnv(ctx context.Context, name string, newEnvironments map[string]*entity.ProjectEnv) error {
	encryptedData, err := s.repo.Get(ctx, entity.BucketProjects, []byte(name))
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if encryptedData == nil {
		return fmt.Errorf("project not found")
	}

	decryptedData, err := crypto.Decrypt(encryptedData, s.masterKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt project data: %w", err)
	}

	var proj entity.Project
	if err := json.Unmarshal(decryptedData, &proj); err != nil {
		return fmt.Errorf("failed to unmarshal project data: %w", err)
	}

	if newEnvironments != nil {
		if proj.Environments == nil {
			proj.Environments = make(map[string]*entity.ProjectEnv)
		}
		for envName, envData := range newEnvironments {
			proj.Environments[envName] = envData
		}
	}

	jsonData, err := json.Marshal(proj)
	if err != nil {
		return fmt.Errorf("failed to marshal updated project: %w", err)
	}

	cryptedData, err := crypto.Encrypt(jsonData, s.masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt updated project: %w", err)
	}

	if err := s.repo.Put(ctx, entity.BucketProjects, []byte(proj.Name), cryptedData); err != nil {
		return fmt.Errorf("failed to save updated project: %w", err)
	}

	return nil
}

// ListProjectsWithEnvs возвращает мапу, где ключ — имя проекта, а значение — список его окружений
func (s *service) ListProjectsWithEnvs(ctx context.Context) (map[string][]string, error) {
	keys, err := s.repo.ListKeys(ctx, entity.BucketProjects)
	if err != nil {
		return nil, fmt.Errorf("failed to list project keys: %w", err)
	}

	result := make(map[string][]string)

	for _, key := range keys {
		proj, err := s.GetProject(ctx, string(key))
		if err != nil {
			continue
		}

		envs := make([]string, 0, len(proj.Environments))
		for envName := range proj.Environments {
			envs = append(envs, envName)
		}

		result[proj.Name] = envs
	}

	return result, nil
}

func (s *service) DeleteProject(ctx context.Context, name string) error {
	return s.repo.Delete(ctx, entity.BucketProjects, []byte(name))
}
