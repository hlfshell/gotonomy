// Package assets provides embedded asset files for the gogentic library.
// Prompt templates and other static assets are embedded at compile time
// using Go's embed directive, making them available without external file access.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/hlfshell/gogentic/pkg/prompt"
)

// Prompts embeds all prompt template files from the prompts directory.
//
//go:embed prompts/*.prompt
var Prompts embed.FS

// PromptNames contains the names of all available embedded prompts.
var PromptNames = []string{
	"planner.prompt",
}

// LoadPrompt loads an embedded prompt template by name.
// The name should include the .prompt extension (e.g., "planner.prompt").
// Returns the loaded template or an error if not found.
func LoadPrompt(name string) (*prompt.Template, error) {
	// Read the embedded file
	content, err := Prompts.ReadFile(filepath.Join("prompts", name))
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded prompt %s: %w", name, err)
	}

	// Add the template to the prompt cache
	tmpl, err := prompt.AddTemplate(name, string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded prompt %s: %w", name, err)
	}

	return tmpl, nil
}

// LoadAllPrompts loads all embedded prompt templates into the default prompt cache.
// This is useful for initializing the system with all available prompts.
func LoadAllPrompts() error {
	return fs.WalkDir(Prompts, "prompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .prompt files
		if filepath.Ext(path) != ".prompt" {
			return nil
		}

		// Load the prompt
		name := filepath.Base(path)
		if _, err := LoadPrompt(name); err != nil {
			return fmt.Errorf("failed to load prompt %s: %w", name, err)
		}

		return nil
	})
}

// GetPromptContent returns the raw content of an embedded prompt without parsing.
// This is useful if you need to inspect or manipulate the prompt content directly.
func GetPromptContent(name string) (string, error) {
	content, err := Prompts.ReadFile(filepath.Join("prompts", name))
	if err != nil {
		return "", fmt.Errorf("failed to read embedded prompt %s: %w", name, err)
	}
	return string(content), nil
}

// ListPrompts returns a list of all available embedded prompt names.
func ListPrompts() ([]string, error) {
	var prompts []string

	err := fs.WalkDir(Prompts, "prompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only include .prompt files
		if filepath.Ext(path) != ".prompt" {
			return nil
		}

		prompts = append(prompts, filepath.Base(path))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	return prompts, nil
}
