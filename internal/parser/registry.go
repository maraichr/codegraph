package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Registry maps file extensions to parsers.
type Registry struct {
	parsers map[string]Parser // extension -> parser
}

func NewRegistry() *Registry {
	return &Registry{parsers: make(map[string]Parser)}
}

func (r *Registry) Register(ext string, p Parser) {
	r.parsers[strings.ToLower(ext)] = p
}

// ForFile returns the parser for a given file path, or nil if none matches.
func (r *Registry) ForFile(path string) Parser {
	ext := strings.ToLower(filepath.Ext(path))
	return r.parsers[ext]
}

// ParseFile detects the parser and parses the file.
func (r *Registry) ParseFile(input FileInput) (*ParseResult, error) {
	p := r.ForFile(input.Path)
	if p == nil {
		return nil, fmt.Errorf("no parser for file: %s", input.Path)
	}
	return p.Parse(input)
}

// SupportedExtensions returns all registered extensions.
func (r *Registry) SupportedExtensions() []string {
	exts := make([]string, 0, len(r.parsers))
	for ext := range r.parsers {
		exts = append(exts, ext)
	}
	return exts
}
