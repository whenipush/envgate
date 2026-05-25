package entity

import (
	"time"
)

type Bucket []byte

var (
	BucketProjects = Bucket("projects")
	BucketTokens   = Bucket("tokens")
)

type ProjectEnv struct {
	Variables map[string]string `json:"variables"`
}

type Project struct {
	Name         string                 `json:"name"`
	Environments map[string]*ProjectEnv `json:"environments"`
}

type TokenMeta struct {
	ProjectName string    `json:"project_name"`
	Environment string    `json:"environment"`
	User        string    `json:"user"`
	CreatedAt   time.Time `json:"created_at"`
}
