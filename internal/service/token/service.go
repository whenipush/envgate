package token

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/whenipush/envgate/internal/entity"
	"github.com/whenipush/envgate/internal/pkg/crypto"
)

type Repository interface {
	Get(ctx context.Context, bucket entity.Bucket, key []byte) ([]byte, error)
	Put(ctx context.Context, bucket entity.Bucket, key []byte, value []byte) error
	Delete(ctx context.Context, bucket entity.Bucket, key []byte) error
	Scan(ctx context.Context, bucket entity.Bucket, cb func(k, v []byte) error) error
}

type ProjectProvider interface {
	GetProject(ctx context.Context, name string) (*entity.Project, error)
}

type service struct {
	repo       Repository
	projectSvc ProjectProvider
	masterKey  []byte
}

func NewService(repo Repository, projectSvc ProjectProvider, masterKey []byte) *service {
	return &service{
		repo:       repo,
		projectSvc: projectSvc,
		masterKey:  masterKey,
	}
}

func (s *service) GenerateToken(ctx context.Context, projectName string, environment string, user string) (string, error) {
	proj, err := s.projectSvc.GetProject(ctx, projectName)
	if err != nil {
		return "", fmt.Errorf("project validation failed: %w", err)
	}
	if proj == nil {
		return "", fmt.Errorf("project '%s' does not exist", projectName)
	}

	if proj.Environments == nil {
		return "", fmt.Errorf("project '%s' has no environments configured", projectName)
	}
	if _, exists := proj.Environments[environment]; !exists {
		return "", fmt.Errorf("environment '%s' not found in project '%s'", environment, projectName)
	}

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}

	tokenStr := "eg_tok_" + hex.EncodeToString(bytes)

	meta := entity.TokenMeta{
		ProjectName: projectName,
		Environment: environment,
		User:        user,
		CreatedAt:   time.Now(),
	}

	plaintext, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}

	encryptedData, err := crypto.Encrypt(plaintext, s.masterKey)
	if err != nil {
		return "", err
	}

	err = s.repo.Put(ctx, entity.BucketTokens, []byte(tokenStr), encryptedData)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (s *service) GetTokenMeta(ctx context.Context, tokenStr string) (*entity.TokenMeta, error) {
	encryptedData, err := s.repo.Get(ctx, entity.BucketTokens, []byte(tokenStr))
	if err != nil {
		return nil, err
	}
	if encryptedData == nil {
		return nil, fmt.Errorf("token not found")
	}

	decryptedData, err := crypto.Decrypt(encryptedData, s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token metadata: %w", err)
	}

	var meta entity.TokenMeta
	if err := json.Unmarshal(decryptedData, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

func (s *service) RevokeToken(ctx context.Context, token string) error {
	return s.repo.Delete(ctx, entity.BucketTokens, []byte(token))
}

func (s *service) ListTokens(ctx context.Context, projectName string) (map[string]*entity.TokenMeta, error) {
	tokens := make(map[string]*entity.TokenMeta)

	err := s.repo.Scan(ctx, entity.BucketTokens, func(k, v []byte) error {
		decryptedData, err := crypto.Decrypt(v, s.masterKey)
		if err != nil {
			return nil
		}

		var meta entity.TokenMeta
		if err := json.Unmarshal(decryptedData, &meta); err != nil {
			return nil
		}

		if meta.ProjectName == projectName {
			tokens[string(k)] = &meta
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return tokens, nil
}
