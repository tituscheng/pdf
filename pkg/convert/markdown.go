package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alnah/picoloom/v2"
)

func init() {
	Register(".md", &Markdown{})
	Register(".markdown", &Markdown{})
}

// Markdown converts Markdown files to PDF using picoloom.
type Markdown struct{}

// Convert reads a Markdown file and writes a PDF to outputPath.
// If opts.CSSPath is set, that CSS file is used directly.
// Otherwise, a sibling CSS file is auto-discovered.
func (m *Markdown) Convert(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(inputPath), err)
	}

	var cssPath string
	if opts != nil {
		cssPath = opts.CSSPath
	}

	css, err := m.resolveCSS(inputPath, cssPath)
	if err != nil {
		return err
	}

	pdf, err := m.convertToPDF(ctx, string(content), filepath.Dir(inputPath), css)
	if err != nil {
		return fmt.Errorf("converting %s: %w", filepath.Base(inputPath), err)
	}

	if err := os.WriteFile(outputPath, pdf, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}
	return nil
}

// resolveCSS returns CSS content. If cssPath is non-empty it reads that file;
// otherwise it auto-discovers a sibling CSS file.
func (m *Markdown) resolveCSS(inputPath, cssPath string) (string, error) {
	if cssPath != "" {
		b, err := os.ReadFile(cssPath)
		if err != nil {
			return "", fmt.Errorf("reading CSS %s: %w", cssPath, err)
		}
		return string(b), nil
	}
	return loadSiblingCSS(inputPath)
}

// convertToPDF renders Markdown + CSS to a PDF byte slice via picoloom.
func (m *Markdown) convertToPDF(ctx context.Context, markdown, sourceDir, css string) (_ []byte, err error) {
	conv, err := picoloom.NewConverter()
	if err != nil {
		return nil, fmt.Errorf("initializing picoloom: %w", err)
	}
	defer func() {
		if closeErr := conv.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing converter: %w", closeErr)
		}
	}()

	result, err := conv.Convert(ctx, picoloom.Input{
		Markdown:  markdown,
		SourceDir: sourceDir,
		CSS:       css,
	})
	if err != nil {
		return nil, err
	}
	return result.PDF, nil
}

// loadSiblingCSS looks for a CSS file next to inputPath. It first tries
// <name>.css, style.css, and styles.css. Failing that, if the directory
// contains exactly one .css file, it uses that. Returns the contents or an
// empty string if none found.
func loadSiblingCSS(inputPath string) (string, error) {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	name := base[:len(base)-len(filepath.Ext(base))]

	candidates := []string{
		filepath.Join(dir, name+".css"),
		filepath.Join(dir, "style.css"),
		filepath.Join(dir, "styles.css"),
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			return string(b), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("reading CSS %s: %w", p, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".css" {
			matches = append(matches, e.Name())
		}
	}
	if len(matches) == 1 {
		p := filepath.Join(dir, matches[0])
		b, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("reading CSS %s: %w", p, err)
		}
		return string(b), nil
	}
	return "", nil
}

// ConvertMarkdown converts a Markdown file to PDF without registry lookup.
// CSS is auto-discovered from sibling files unless opts.CSSPath is set.
func ConvertMarkdown(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	return (&Markdown{}).Convert(ctx, inputPath, outputPath, opts)
}

// ConvertMarkdownToPDF converts Markdown bytes to PDF bytes using picoloom.
// css is optional; pass an empty string for no custom styling.
func ConvertMarkdownToPDF(ctx context.Context, markdown []byte, css string) ([]byte, error) {
	return (&Markdown{}).convertToPDF(ctx, string(markdown), "", css)
}
