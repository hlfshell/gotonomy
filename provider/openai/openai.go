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
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

// Constants
const (
	defaultBaseURL     = "https://api.openai.com/v1"
	defaultTimeoutSecs = 30
	defaultMaxRetries  = 3
)

// OpenAI implements the provider.Provider interface for OpenAI.
type OpenAI struct {
	client     openai.Client
	modelCards map[string]model.ModelDescription // Cache of loaded model cards keyed by model name
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

	provider := &OpenAI{
		client:     client,
		modelCards: make(map[string]model.ModelDescription),
	}

	// Load model cards from the provider's directory
	if err := provider.loadModelCards(); err != nil {
		// Log but don't fail - model cards are optional, we can still use API
		// In production, you might want to use a logger here
		_ = err
	}

	return provider, nil
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

// loadModelCards loads model cards from the provider's models.yaml file.
// It uses the default paths which will automatically find provider model cards.
// TODO: Implement model card loading when model.LoadDefaultPaths() is available
func (p *OpenAI) loadModelCards() error {
	// Model card loading is temporarily disabled until model.LoadDefaultPaths() is implemented
	// The provider will still work by discovering models via the API
	return nil
}

// ListAvailableModels returns a list of available models from OpenAI.
// It uses model cards as the primary source, with API results as a fallback.
func (p *OpenAI) ListAvailableModels(ctx context.Context) ([]model.ModelDescription, error) {
	// Start with models from model cards
	modelDescriptions := make([]model.ModelDescription, 0, len(p.modelCards))
	for _, desc := range p.modelCards {
		modelDescriptions = append(modelDescriptions, desc)
	}

	// Optionally merge with API results for models not in cards
	// This allows discovering new models via API while using cards for known models
	apiModels, err := p.client.Models.List(ctx)
	if err != nil {
		// If API call fails but we have model cards, return those
		if len(modelDescriptions) > 0 {
			return modelDescriptions, nil
		}
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	// Add API models that aren't already in our cards
	for _, m := range apiModels.Data {
		// Only include chat models
		if strings.Contains(m.ID, "gpt") {
			// Check if we already have this model from cards
			if _, exists := p.modelCards[m.ID]; exists {
				continue
			}

			// Create a basic ModelDescription for API-discovered models
			info := model.ModelDescription{
				Model:       m.ID,
				Provider:    "openai",
				Description: fmt.Sprintf("OpenAI %s model", m.ID),
				CanUseTools: true,
			}

			// Set context window size based on model (fallback defaults)
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

			// Set default costs (should be updated in model cards)
			info.Costs = model.CostsPerToken{
				Input:  0.00001,
				Output: 0.00003,
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
	// First check model cards cache
	modelInfo, found := p.modelCards[modelName]
	if !found {
		// Fallback to listing from API
		models, err := p.ListAvailableModels(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list models: %w", err)
		}

		found = false
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
	}

	// Create and return the model
	return &OpenAIModel{
		provider:  p,
		modelInfo: modelInfo,
	}, nil
}

// AddModel allows you to add a model instance by ModelDescription.
// This is useful for adding custom models or models discovered at runtime.
func (p *OpenAI) AddModel(ctx context.Context, modelDesc model.ModelDescription) error {
	// Validate the model description
	if err := modelDesc.Validate(); err != nil {
		return fmt.Errorf("invalid model description: %w", err)
	}

	// Ensure the provider matches
	if modelDesc.Provider != "openai" {
		return fmt.Errorf("model provider %s does not match OpenAI provider", modelDesc.Provider)
	}

	// Add to cache
	p.modelCards[modelDesc.Model] = modelDesc
	return nil
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
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(request.Messages))
	for _, msg := range request.Messages {
		var messageUnion openai.ChatCompletionMessageParamUnion

		// Content is a union type - we'll use OfString for simple text content
		contentUnion := openai.ChatCompletionUserMessageParamContentUnion{
			OfString: param.NewOpt(msg.Content),
		}

		switch msg.Role {
		case model.RoleSystem:
			systemContent := openai.ChatCompletionSystemMessageParamContentUnion{
				OfString: param.NewOpt(msg.Content),
			}
			messageUnion = openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: systemContent,
				},
			}
		case model.RoleUser:
			messageUnion = openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: contentUnion,
				},
			}
		case model.RoleAssistant:
			assistantContent := openai.ChatCompletionAssistantMessageParamContentUnion{
				OfString: param.NewOpt(msg.Content),
			}
			messageUnion = openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content: assistantContent,
				},
			}
		case model.RoleTool:
			// Tool messages need a tool_call_id to match the original tool call
			if msg.ToolCallID == "" {
				// Skip tool messages without a tool call ID
				continue
			}
			toolContent := openai.ChatCompletionToolMessageParamContentUnion{
				OfString: param.NewOpt(msg.Content),
			}
			messageUnion = openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					Content:    toolContent,
					ToolCallID: msg.ToolCallID,
				},
			}
		default:
			return model.CompletionResponse{}, fmt.Errorf("unsupported message role: %s", msg.Role)
		}

		openaiMessages = append(openaiMessages, messageUnion)
	}

	// Build the request
	chatParams := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(m.modelInfo.Model),
		Messages: openaiMessages,
	}

	// Set temperature from config
	if request.Config.Temperature > 0 {
		chatParams.Temperature = param.NewOpt(float64(request.Config.Temperature))
	}

	// Convert tools if provided
	if len(request.Tools) > 0 {
		openaiTools := make([]openai.ChatCompletionToolUnionParam, 0, len(request.Tools))
		for _, t := range request.Tools {
			// Convert parameters to JSON schema
			params := t.Parameters()
			schema := tool.ParametersToJSONSchema(params)

			// Convert schema to map[string]any for Parameters field
			var schemaMap map[string]any
			if schemaBytes, err := json.Marshal(schema); err == nil {
				json.Unmarshal(schemaBytes, &schemaMap)
			}

			// Convert schema map to the proper Parameters type
			// FunctionParameters is typically map[string]any or a JSON schema
			functionTool := openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        t.Name(),
					Description: param.NewOpt(t.Description()),
					Parameters:  schemaMap,
				},
			}

			openaiTools = append(openaiTools, openai.ChatCompletionToolUnionParam{
				OfFunction: &functionTool,
			})
		}
		chatParams.Tools = openaiTools
	}

	// Make the request
	completion, err := m.provider.client.Chat.Completions.New(ctx, chatParams)
	if err != nil {
		return model.CompletionResponse{}, fmt.Errorf("failed to create completion: %w", err)
	}

	// Convert the response
	if len(completion.Choices) == 0 {
		return model.CompletionResponse{}, errors.New("no choices in response")
	}

	choice := completion.Choices[0]

	// Extract text content from the message
	// In the response, Content is a simple string
	textContent := choice.Message.Content

	genericResponse := model.CompletionResponse{
		Text: textContent,
		UsageStats: model.UsageStats{
			InputTokens:  int(completion.Usage.PromptTokens),
			OutputTokens: int(completion.Usage.CompletionTokens),
		},
	}

	// Convert tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]model.ToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			// Parse the arguments JSON
			var args tool.Arguments
			if tc.Function.Arguments != "" {
				// Arguments is a string containing JSON
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return model.CompletionResponse{}, fmt.Errorf("failed to parse tool call arguments: %w", err)
				}
			}

			toolCall := model.ToolCall{
				ID:        tc.ID, // Store the tool call ID from OpenAI
				Name:      tc.Function.Name,
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
	// Input is a union type - use OfArrayOfStrings for multiple texts
	inputUnion := openai.EmbeddingNewParamsInputUnion{
		OfArrayOfStrings: texts,
	}
	embedParams := openai.EmbeddingNewParams{
		Model: m.modelInfo.Name, // Model is just a string
		Input: inputUnion,
	}

	// Make the request
	embeddings, err := m.provider.client.Embeddings.New(ctx, embedParams)
	if err != nil {
		return embedding.EmbeddingResponse{}, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Convert the response to the generic format
	resultEmbeddings := make([]embedding.Embedding, 0, len(embeddings.Data))
	for i, data := range embeddings.Data {
		// Convert []float64 to []float32 if needed
		vector := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vector[j] = float32(v)
		}
		resultEmbeddings = append(resultEmbeddings, embedding.Embedding{
			Vector: vector,
			Index:  i,
		})
	}

	return embedding.EmbeddingResponse{
		Embeddings: resultEmbeddings,
		UsageStats: embedding.UsageStats{
			TokensProcessed: int(embeddings.Usage.PromptTokens),
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
