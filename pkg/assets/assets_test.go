package assets

import (
	"strings"
	"testing"
)

func TestLoadPrompt(t *testing.T) {
	// Test loading the planner prompt
	tmpl, err := LoadPrompt("planner.prompt")
	if err != nil {
		t.Fatalf("Failed to load planner.prompt: %v", err)
	}

	if tmpl == nil {
		t.Fatal("Template is nil")
	}

	if tmpl.Name != "planner.prompt" {
		t.Errorf("Expected template name 'planner.prompt', got '%s'", tmpl.Name)
	}

	if tmpl.Content == "" {
		t.Error("Template content is empty")
	}

	// Verify the content contains expected sections
	if !strings.Contains(tmpl.Content, "OBJECTIVE") {
		t.Error("Template should contain OBJECTIVE section")
	}

	if !strings.Contains(tmpl.Content, "PLAN FORMAT REQUIREMENTS") {
		t.Error("Template should contain PLAN FORMAT REQUIREMENTS section")
	}
}

func TestLoadPrompt_NotFound(t *testing.T) {
	_, err := LoadPrompt("nonexistent.prompt")
	if err == nil {
		t.Error("Expected error when loading non-existent prompt")
	}
}

func TestLoadAllPrompts(t *testing.T) {
	err := LoadAllPrompts()
	if err != nil {
		t.Fatalf("Failed to load all prompts: %v", err)
	}

	// Verify at least the planner prompt was loaded
	prompts, err := ListPrompts()
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}

	if len(prompts) == 0 {
		t.Error("Expected at least one prompt to be loaded")
	}

	found := false
	for _, name := range prompts {
		if name == "planner.prompt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("planner.prompt should be in the list of prompts")
	}
}

func TestGetPromptContent(t *testing.T) {
	content, err := GetPromptContent("planner.prompt")
	if err != nil {
		t.Fatalf("Failed to get prompt content: %v", err)
	}

	if content == "" {
		t.Error("Prompt content is empty")
	}

	// Verify content contains expected elements
	if !strings.Contains(content, "objective") {
		t.Error("Content should contain 'objective' variable")
	}

	if !strings.Contains(content, "steps") {
		t.Error("Content should contain 'steps' reference")
	}
}

func TestGetPromptContent_NotFound(t *testing.T) {
	_, err := GetPromptContent("missing.prompt")
	if err == nil {
		t.Error("Expected error when getting content of non-existent prompt")
	}
}

func TestListPrompts(t *testing.T) {
	prompts, err := ListPrompts()
	if err != nil {
		t.Fatalf("Failed to list prompts: %v", err)
	}

	if len(prompts) == 0 {
		t.Error("Expected at least one prompt")
	}

	// All prompt names should end with .prompt
	for _, name := range prompts {
		if !strings.HasSuffix(name, ".prompt") {
			t.Errorf("Prompt name '%s' should end with .prompt", name)
		}
	}

	// Should include planner.prompt
	found := false
	for _, name := range prompts {
		if name == "planner.prompt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected planner.prompt to be in the list")
	}
}

func TestPromptNames(t *testing.T) {
	if len(PromptNames) == 0 {
		t.Error("PromptNames should not be empty")
	}

	// Check that all names in PromptNames are actually available
	for _, name := range PromptNames {
		_, err := LoadPrompt(name)
		if err != nil {
			t.Errorf("Prompt %s listed in PromptNames but failed to load: %v", name, err)
		}
	}
}

func TestEmbeddedPromptParsing(t *testing.T) {
	// Load the template
	tmpl, err := LoadPrompt("planner.prompt")
	if err != nil {
		t.Fatalf("Failed to load prompt: %v", err)
	}

	// Test rendering with sample data
	data := map[string]interface{}{
		"objective": "Test objective",
		"tools":     []map[string]string{},
	}

	rendered, err := tmpl.Render(data)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	if rendered == "" {
		t.Error("Rendered output is empty")
	}

	// Verify the objective was inserted
	if !strings.Contains(rendered, "Test objective") {
		t.Error("Rendered output should contain the objective")
	}
}

