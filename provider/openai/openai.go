// Package openai provides an implementation of the provider interface for OpenAI.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hlfshell/gotonomy/embedding"
	"github.com/hlfshell/gotonomy/model"
	"github.com/hlfshell/gotonomy/provider"
	"github.com/hlfshell/gotonomy/tool"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Constants
const (
	defaultBaseURL     = "https://api.openai.com/v1"
	defaultTimeoutSecs = 30
	defaultMaxRetries  = 3
)

// OpenAI implements the provider.Provider interface for OpenAI.
type OpenAI struct {
	client *openai.Client
}

// NewOpenAIProvider creates a new OpenAI provider with the given configuration.
func NewOpenAIProvider(config provider.Config) (provider.Provider, error) {
	// Validate the API key
	if config.APIKey == "" {
		return nil, errors.New("API key is required for OpenAI provider")
	}

	// Build options
	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	// Set base URL if provided
	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	// Set organization ID if provided
	if config.OrganizationID != "" {
		opts = append(opts, option.WithOrganization(config.OrganizationID))
	}

	// Create the OpenAI client
	client := openai.NewClient(opts...)

	return &OpenAI{
		client: client,
	}, nil
}

// Name returns the name of the provider.
func (p *OpenAI) Name() string {
	return "OpenAI"
}

// Description returns a human-readable description of the provider.
func (p *OpenAI) Description() string {
	return "OpenAI provides various language and vision-language models including GPT-4o, GPT-4, and GPT-3.5 Turbo."
}

// DefaultConfig returns the default configuration for the provider.
func (p *OpenAI) DefaultConfig() provider.Config {
	return provider.Config{
		BaseURL:        defaultBaseURL,
		TimeoutSeconds: defaultTimeoutSecs,
		MaxRetries:     defaultMaxRetries,
	}
}

// ListAvailableModels returns a list of available models from OpenAI.
func (p *OpenAI) ListAvailableModels(ctx context.Context) ([]model.ModelDescription, error) {
	// List models using the OpenAI client
	models, err := p.client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// Filter and map the models to ModelDescription
	modelDescriptions := []model.ModelDescription{}
	for _, m := range models.Data {
		// Only include chat models
		if strings.Contains(m.ID, "gpt") {
			info := model.ModelDescription{
				Model:       m.ID,
				Provider:    "openai",
				Description: fmt.Sprintf("OpenAI %s model", m.ID),
				CanUseTools: true,
			}

			// Set context window size based on model
			switch {
			case strings.Contains(m.ID, "gpt-4o"):
				info.MaxContextTokens = 128000
			case strings.Contains(m.ID, "gpt-4-turbo"):
				info.MaxContextTokens = 128000
			case strings.Contains(m.ID, "gpt-4-32k"):
				info.MaxContextTokens = 32768
			case strings.Contains(m.ID, "gpt-4"):
				info.MaxContextTokens = 8192
			case strings.Contains(m.ID, "gpt-3.5-turbo-16k"):
				info.MaxContextTokens = 16384
			case strings.Contains(m.ID, "gpt-3.5-turbo"):
				info.MaxContextTokens = 4096
			default:
				info.MaxContextTokens = 4096
			}

			// Set costs (example values - should be updated with actual pricing)
			info.Costs = model.CostsPerToken{
				Input:  0.00001, // $0.01 per 1M tokens
				Output: 0.00003, // $0.03 per 1M tokens
			}

			modelDescriptions = append(modelDescriptions, info)
		}
	}

	return modelDescriptions, nil
}

// ListAvailableEmbeddingModels returns a list of available embedding models from OpenAI.
func (p *OpenAI) ListAvailableEmbeddingModels(ctx context.Context) ([]embedding.ModelInfo, error) {
	// Define the available embedding models
	embeddingModels := []embedding.ModelInfo{
		{
			Name:                  "text-embedding-3-large",
			Provider:              "openai",
			Dimensions:            3072,
			SupportedContentTypes: []embedding.ContentType{embedding.TextContent},
			Description:           "OpenAI's most capable embedding model for text",
		},
		{
			Name:                  "text-embedding-3-small",
			Provider:              "openai",
			Dimensions:            1536,
			SupportedContentTypes: []embedding.ContentType{embedding.TextContent},
			Description:           "OpenAI's efficient embedding model for text with good quality",
		},
		{
			Name:                  "text-embedding-ada-002",
			Provider:              "openai",
			Dimensions:            1536,
			SupportedContentTypes: []embedding.ContentType{embedding.TextContent},
			Description:           "OpenAI's legacy embedding model (deprecated)",
		},
	}

	return embeddingModels, nil
}

// GetModel returns a model instance by name.
func (p *OpenAI) GetModel(ctx context.Context, modelName string) (model.Model, error) {
	// Check if the model exists
	models, err := p.ListAvailableModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	var modelInfo model.ModelDescription
	found := false
	for _, info := range models {
		if info.Model == modelName {
			modelInfo = info
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("model %s not found", modelName)
	}

	// Create and return the model
	return &OpenAIModel{
		provider:  p,
		modelInfo: modelInfo,
	}, nil
}

// GetEmbeddingModel returns an embedding model instance by name.
func (p *OpenAI) GetEmbeddingModel(ctx context.Context, modelName string) (embedding.EmbeddingModel, error) {
	// Check if the model exists
	models, err := p.ListAvailableEmbeddingModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list embedding models: %w", err)
	}

	var modelInfo embedding.ModelInfo
	found := false
	for _, info := range models {
		if info.Name == modelName {
			modelInfo = info
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("embedding model %s not found", modelName)
	}

	// Create and return the embedding model
	return &OpenAIEmbeddingModel{
		provider:  p,
		modelInfo: modelInfo,
	}, nil
}

// OpenAIModel implements the model.Model interface for OpenAI models.
type OpenAIModel struct {
	provider  *OpenAI
	modelInfo model.ModelDescription
}

// Description returns information about the model.
func (m *OpenAIModel) Description() model.ModelDescription {
	return m.modelInfo
}

// Complete generates a completion for the given request.
func (m *OpenAIModel) Complete(ctx context.Context, request model.CompletionRequest) (model.CompletionResponse, error) {
	// Validate the request
	if err := request.Validate(); err != nil {
		return model.CompletionResponse{}, fmt.Errorf("invalid request: %w", err)
	}

	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessageParam, 0, len(request.Messages))
	for _, msg := range request.Messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessageParam{
			Role:    openai.F(openai.ChatCompletionMessageRole(msg.Role)),
			Content: openai.F(msg.Content),
		})
	}

	// Build the request
	chatParams := openai.ChatCompletionCreateParams{
		Model:    openai.F(m.modelInfo.Model),
		Messages: openaiMessages,
	}

	// Set temperature from config
	if request.Config.Temperature > 0 {
		chatParams.Temperature = openai.F(request.Config.Temperature)
	}

	// Convert tools if provided
	if len(request.Tools) > 0 {
		openaiTools := make([]openai.ChatCompletionToolParam, 0, len(request.Tools))
		for _, t := range request.Tools {
			// Convert parameters to JSON schema
			params := t.Parameters()
			schema := tool.ParametersToJSONSchema(params)

			openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
				Type: openai.F(openai.ChatCompletionToolTypeFunction),
				Function: openai.F(openai.FunctionDefinition{
					Name:        openai.F(t.Name()),
					Description: openai.F(t.Description()),
					Parameters:  openai.F(schema),
				}),
			})
		}
		chatParams.Tools = openai.F(openaiTools)
	}

	// Make the request
	completion, err := m.provider.client.Chat.Completions.Create(ctx, chatParams)
	if err != nil {
		return model.CompletionResponse{}, fmt.Errorf("failed to create completion: %w", err)
	}

	// Convert the response
	if len(completion.Choices) == 0 {
		return model.CompletionResponse{}, errors.New("no choices in response")
	}

	choice := completion.Choices[0]
	genericResponse := model.CompletionResponse{
		Text: choice.Message.Content[0].Text,
		UsageStats: model.UsageStats{
			InputTokens:  completion.Usage.PromptTokens,
			OutputTokens: completion.Usage.CompletionTokens,
			// OpenAI doesn't separate reasoning tokens
			ReasoningTokens: 0,
		},
	}

	// Convert tool calls if present
	if choice.Message.ToolCalls != nil && len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]model.ToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			// Parse the arguments JSON
			var args tool.Arguments
			if tc.Function.Arguments != nil {
				if err := json.Unmarshal([]byte(*tc.Function.Arguments), &args); err != nil {
					return model.CompletionResponse{}, fmt.Errorf("failed to parse tool call arguments: %w", err)
				}
			}

			toolCall := model.ToolCall{
				Name:      *tc.Function.Name,
				Arguments: args,
			}
			toolCalls = append(toolCalls, toolCall)
		}
		genericResponse.ToolCalls = toolCalls
	}

	return genericResponse, nil
}

// OpenAIEmbeddingModel implements the embedding.EmbeddingModel interface for OpenAI.
type OpenAIEmbeddingModel struct {
	provider  *OpenAI
	modelInfo embedding.ModelInfo
}

// GetInfo returns information about the embedding model.
func (m *OpenAIEmbeddingModel) GetInfo() embedding.ModelInfo {
	return m.modelInfo
}

// Embed generates embeddings for the given request.
func (m *OpenAIEmbeddingModel) Embed(ctx context.Context, request embedding.EmbeddingRequest) (embedding.EmbeddingResponse, error) {
	// Check if all content types are supported
	for _, content := range request.Contents {
		if !m.SupportsContentType(content.Type) {
			return embedding.EmbeddingResponse{}, fmt.Errorf("content type %s not supported by model %s", content.Type, m.modelInfo.Name)
		}
	}

	// Extract text from the request
	texts := make([]string, 0, len(request.Contents))
	for _, content := range request.Contents {
		if content.Type == embedding.TextContent {
			texts = append(texts, content.Text)
		} else {
			return embedding.EmbeddingResponse{}, fmt.Errorf("non-text content not supported yet")
		}
	}

	// Create the embedding request
	embedParams := openai.EmbeddingCreateParams{
		Model: openai.F(m.modelInfo.Name),
		Input: openai.F(texts),
	}

	// Make the request
	embeddings, err := m.provider.client.Embeddings.Create(ctx, embedParams)
	if err != nil {
		return embedding.EmbeddingResponse{}, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Convert the response to the generic format
	resultEmbeddings := make([]embedding.Embedding, 0, len(embeddings.Data))
	for i, data := range embeddings.Data {
		resultEmbeddings = append(resultEmbeddings, embedding.Embedding{
			Vector: data.Embedding,
			Index:  i,
		})
	}

	return embedding.EmbeddingResponse{
		Embeddings: resultEmbeddings,
		UsageStats: embedding.UsageStats{
			TokensProcessed: embeddings.Usage.PromptTokens,
		},
	}, nil
}

// SupportsContentType checks if the model supports a specific content type.
func (m *OpenAIEmbeddingModel) SupportsContentType(contentType embedding.ContentType) bool {
	for _, supported := range m.modelInfo.SupportedContentTypes {
		if supported == contentType {
			return true
		}
	}
	return false
}
