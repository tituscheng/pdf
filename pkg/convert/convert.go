// Package convert provides a modular and extensible way to convert
// various file formats to PDF.
//
// Basic usage via the default registry:
//
//	c, err := convert.For(".md")
//	if err != nil {
//	    // handle unsupported format
//	}
//	if err := c.Convert(ctx, "input.md", "output.pdf", nil); err != nil {
//	    // handle conversion error
//	}
//
// Or use the high-level convenience function:
//
//	if err := convert.ConvertFile(ctx, "input.md", "output.pdf", nil); err != nil {
//	    // handle error
//	}
//
// For isolated registries (e.g., testing, concurrent use with different options):
//
//	reg := convert.NewRegistry()
//	reg.Register(".md", &convert.Markdown{})
//	reg.ConvertFile(ctx, "input.md", "output.pdf", &convert.Options{CSSPath: "style.css"})
package convert

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ErrUnsupportedFormat is returned when no converter is registered for a
// file extension.
var ErrUnsupportedFormat = errors.New("unsupported file format")

// Options holds per-conversion configuration.
type Options struct {
	// CSSPath is the path to a custom CSS file. When empty, converters that
	// support CSS will attempt auto-discovery.
	CSSPath string
}

// Converter defines the interface for file-to-PDF conversion.
type Converter interface {
	// Convert reads the file at inputPath and writes a PDF to outputPath.
	// opts may be nil; converters should treat nil as "use defaults".
	Convert(ctx context.Context, inputPath, outputPath string, opts *Options) error
}

// Named is implemented by converters that expose a human-readable name.
type Named interface {
	Name() string
}

// Registry maps file extensions to Converter implementations.
type Registry struct {
	mu       sync.RWMutex
	registry map[string]Converter
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		registry: make(map[string]Converter),
	}
}

// Register associates a file extension (e.g., ".md") with a Converter.
// The extension should include the leading dot and be lower-case.
func (r *Registry) Register(ext string, c Converter) {
	ext = normalizeExt(ext)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registry[ext] = c
}

// For returns the Converter registered for the given file extension.
// The extension is normalised before lookup.
func (r *Registry) For(ext string) (Converter, error) {
	ext = normalizeExt(ext)
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.registry[ext]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedFormat, ext)
	}
	return c, nil
}

// SupportedExtensions returns a sorted slice of all registered extensions.
func (r *Registry) SupportedExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exts := make([]string, 0, len(r.registry))
	for ext := range r.registry {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// ConvertFile converts a single file to PDF using the registry.
// The appropriate converter is selected based on the file extension.
func (r *Registry) ConvertFile(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	ext := filepath.Ext(inputPath)
	c, err := r.For(ext)
	if err != nil {
		return err
	}
	return c.Convert(ctx, inputPath, outputPath, opts)
}

func normalizeExt(ext string) string {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

// DefaultOutputPath returns a sensible PDF output path for the given input.
// If inputPath is "/docs/report.md" the result is "/docs/report.pdf".
func DefaultOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	return base + ".pdf"
}

// ---------------------------------------------------------------------------
// Default registry and convenience wrappers
// ---------------------------------------------------------------------------

// Default is the package-level registry used by the convenience functions.
// Library callers who need isolation (e.g., concurrent use with different
// configurations) should create their own Registry with NewRegistry.
var Default = NewRegistry()

// Register registers a converter on the default registry.
func Register(ext string, c Converter) { Default.Register(ext, c) }

// For looks up a converter in the default registry.
func For(ext string) (Converter, error) { return Default.For(ext) }

// SupportedExtensions returns extensions from the default registry.
func SupportedExtensions() []string { return Default.SupportedExtensions() }

// ConvertFile converts a file using the default registry.
func ConvertFile(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	return Default.ConvertFile(ctx, inputPath, outputPath, opts)
}
