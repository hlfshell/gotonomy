package template

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/hlfshell/gogentic/agent"
)

// RenderTemplate is the function signature for prompt template functions.
type RenderTemplate func(prompt string, args agent.Arguments) (map[string]string, error)

// templateRegistrty is a thread-safe global registry for embedded templates.
// Templates are stored as functions that wrap parsed templates for efficient rendering.
// The parsed templates are kept in memory privately.
type templateRegistrty struct {
	// functions stores the registered template functions
	functions map[string]RenderTemplate
	// templates stores the parsed templates privately for internal use
	templates map[string]*template.Template
	mutex     sync.RWMutex
}

// globalRegistry is the singleton instance of the template registry.
var globalRegistry = &templateRegistrty{
	functions: make(map[string]RenderTemplate),
	templates: make(map[string]*template.Template),
	mutex:     sync.RWMutex{},
}

// RegisterTemplate registers a template in the global registry by parsing the content
// and creating a function that wraps it. The parsed template is stored privately,
// and a function matching SimplePromptTemplate's signature is stored in the registry.
// If a template with the same name already exists, an error is returned.
func RegisterTemplate(name, content string) error {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %q: %w", name, err)
	}

	// Create a function that wraps the template
	templateFunc := func(prompt string, args agent.Arguments) (map[string]string, error) {
		var buf strings.Builder
		if err := tmpl.Execute(&buf, args); err != nil {
			return nil, fmt.Errorf("failed to render template %q: %w", name, err)
		}
		return map[string]string{
			"prompt": buf.String(),
		}, nil
	}

	globalRegistry.mutex.Lock()
	defer globalRegistry.mutex.Unlock()
	if _, ok := globalRegistry.functions[name]; ok {
		return fmt.Errorf("template %q already registered", name)
	}
	globalRegistry.functions[name] = templateFunc
	globalRegistry.templates[name] = tmpl
	return nil
}

// LoadAndRegisterTemplate loads a template from a file path and registers it
// in the global registry. The file content is read and parsed before registration.
func LoadAndRegisterTemplate(name string, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file %q: %w", path, err)
	}
	return RegisterTemplate(name, string(content))
}

// GetTemplate retrieves a template function from the global registry by name.
// Returns the function and true if found, nil and false otherwise.
// The function matches SimplePromptTemplate's signature.
func GetTemplate(name string) (RenderTemplate, bool) {
	globalRegistry.mutex.RLock()
	defer globalRegistry.mutex.RUnlock()
	fn, ok := globalRegistry.functions[name]
	return fn, ok
}

// ListTemplates returns a list of all registered template names.
func ListTemplates() []string {
	globalRegistry.mutex.RLock()
	defer globalRegistry.mutex.RUnlock()
	names := make([]string, 0, len(globalRegistry.functions))
	for name := range globalRegistry.functions {
		names = append(names, name)
	}
	return names
}
