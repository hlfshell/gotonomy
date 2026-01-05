// Package provider defines interfaces and implementations for various
// LLM/VLM providers such as OpenAI, Google, Anthropic, etc.
package provider

import (
	"context"

	"github.com/hlfshell/gotonomy/embedding"
	"github.com/hlfshell/gotonomy/model"
)

// Provider represents a service provider for AI models.
type Provider interface {
	// Name returns the name of the provider.
	Name() string

	// Description returns a human-readable description of the provider.
	Description() string

	// ListAvailableModels returns a list of available models from this provider.
	ListAvailableModels(ctx context.Context) ([]model.ModelDescription, error)

	// ListAvailableEmbeddingModels returns a list of available embedding models.
	ListAvailableEmbeddingModels(ctx context.Context) ([]embedding.ModelInfo, error)

	// GetModel returns a model instance by name.
	GetModel(ctx context.Context, modelName string) (model.Model, error)

	// AddModel allows you to add a model instance by ModelDescription to prevent
	// being limited to hard coded models
	AddModel(ctx context.Context, model model.ModelDescription) error

	// GetEmbeddingModel returns an embedding model instance by name.
	GetEmbeddingModel(ctx context.Context, modelName string) (embedding.EmbeddingModel, error)

	// DefaultConfig returns the default configuration for the provider
	DefaultConfig() Config
}

// Config represents the configuration for a provider.
type Config struct {
	APIKey            string            `json:"api_key"`
	BaseURL           string            `json:"base_url"`
	OrganizationID    string            `json:"organization_id"`
	TimeoutSeconds    int               `json:"timeout_seconds"`
	MaxRetries        int               `json:"max_retries"`
	AdditionalHeaders map[string]string `json:"additional_headers"`
}
