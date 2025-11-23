package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hlfshell/gogentic/agent"
)

func main() {
	var (
		typeName    = flag.String("type", "", "Type name to generate tool wrapper for")
		packageName = flag.String("package", "", "Package name (defaults to current package)")
		outputFile  = flag.String("output", "", "Output file (defaults to <type>_tool_gen.go)")
	)
	flag.Parse()

	if *typeName == "" {
		fmt.Fprintln(os.Stderr, "Error: -type flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Get the file path from environment variable or current directory
	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		fmt.Fprintln(os.Stderr, "Error: GOFILE environment variable not set. Run with go generate.")
		os.Exit(1)
	}

	// Get package name from environment or flag
	pkg := *packageName
	if pkg == "" {
		pkg = os.Getenv("GOPACKAGE")
		if pkg == "" {
			pkg = "main"
		}
	}

	// Generate the tool wrapper
	code, err := agent.GenerateToolWrapper(*typeName, pkg, goFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating tool wrapper: %v\n", err)
		os.Exit(1)
	}

	// Determine output file name
	outFile := *outputFile
	if outFile == "" {
		base := strings.TrimSuffix(filepath.Base(goFile), ".go")
		outFile = fmt.Sprintf("%s_tool_gen.go", strings.ToLower(*typeName))
		if base != "" {
			outFile = filepath.Join(filepath.Dir(goFile), outFile)
		}
	}

	// Write the generated code
	if err := os.WriteFile(outFile, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated tool wrapper: %s\n", outFile)
}

