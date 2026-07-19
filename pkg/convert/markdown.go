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

// Markdown converts Markdown files and in-memory Markdown content to PDF
// using picoloom.
type Markdown struct{}

// Convert reads a Markdown file and writes a PDF to outputPath.
//
// CSS resolution (path mode): CSS (inline) > CSSPath > loadSiblingCSS(inputPath).
// Sibling discovery uses the real input path (e.g. report.md prefers report.css)
// and is independent of opts.SourceDir.
//
// Asset base: opts.SourceDir if set, otherwise filepath.Dir(inputPath).
// opts may be nil. Convert does not mutate opts.
func (m *Markdown) Convert(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Base(inputPath), err)
	}

	css, err := resolveCSSContentForPath(inputPath, opts)
	if err != nil {
		return err
	}
	sourceDir := effectiveSourceDir(inputPath, opts)

	pdf, err := m.convertToPDF(ctx, string(content), sourceDir, css)
	if err != nil {
		return fmt.Errorf("converting %s: %w", filepath.Base(inputPath), err)
	}
	if err := os.WriteFile(outputPath, pdf, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}
	return nil
}

// ConvertContent implements ContentConverter. It renders Markdown bytes to PDF
// without reading an input file path or writing an output file.
//
// CSS resolution (content mode): CSS (inline) > CSSPath >
// loadSiblingCSSUnnamed(SourceDir) when SourceDir is set > empty.
// opts may be nil. ConvertContent does not mutate opts.
func (m *Markdown) ConvertContent(ctx context.Context, content []byte, opts *Options) ([]byte, error) {
	css, err := resolveCSSContentForContent(opts)
	if err != nil {
		return nil, err
	}
	sourceDir := effectiveSourceDir("", opts)
	// Markdown text is treated as UTF-8 (same as path mode after ReadFile).
	return m.convertToPDF(ctx, string(content), sourceDir, css)
}

// convertToPDF renders Markdown + CSS to a PDF byte slice via picoloom.
// Pure core: no Options, no discovery, no input-file I/O.
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

// ---------------------------------------------------------------------------
// Resolve helpers (unexported; pure with respect to *Options)
// ---------------------------------------------------------------------------

// effectiveSourceDir returns the picoloom asset base directory.
//
//   - If opts != nil && opts.SourceDir != "", return opts.SourceDir
//   - Else if inputPath != "", return filepath.Dir(inputPath)
//   - Else return "" (content mode with no SourceDir)
//
// Never mutates opts. nil opts is treated as empty SourceDir.
func effectiveSourceDir(inputPath string, opts *Options) string {
	if opts != nil && opts.SourceDir != "" {
		return opts.SourceDir
	}
	if inputPath != "" {
		return filepath.Dir(inputPath)
	}
	return ""
}

// contentCSSDiscoveryDir returns the directory for content-mode CSS discovery,
// or "" if discovery must not run.
func contentCSSDiscoveryDir(opts *Options) string {
	if opts != nil && opts.SourceDir != "" {
		return opts.SourceDir
	}
	return ""
}

// resolveCSSContentForPath is the path-mode CSS entry:
// CSS > CSSPath > loadSiblingCSS(inputPath) > "".
// Real basename: report.md → report.css. Independent of opts.SourceDir.
// Never mutates opts. nil opts means sibling discovery only.
func resolveCSSContentForPath(inputPath string, opts *Options) (string, error) {
	if opts != nil && opts.CSS != "" {
		return opts.CSS, nil
	}
	if opts != nil && opts.CSSPath != "" {
		return readCSSFile(opts.CSSPath)
	}
	return loadSiblingCSS(inputPath)
}

// resolveCSSContentForContent is the content-mode CSS entry:
// CSS > CSSPath > loadSiblingCSSUnnamed(SourceDir) if SourceDir set > "".
// nil opts ⇒ no CSS. Never mutates opts. Does not invent a synthetic .md path.
func resolveCSSContentForContent(opts *Options) (string, error) {
	if opts != nil && opts.CSS != "" {
		return opts.CSS, nil // raw != ""; no trim; do not open CSSPath
	}
	if opts != nil && opts.CSSPath != "" {
		return readCSSFile(opts.CSSPath) // CWD-relative unless absolute
	}
	if dir := contentCSSDiscoveryDir(opts); dir != "" {
		return loadSiblingCSSUnnamed(dir)
	}
	return "", nil
}

// readCSSFile reads a CSS file from cssPath (CWD-relative unless absolute).
func readCSSFile(cssPath string) (string, error) {
	b, err := os.ReadFile(cssPath)
	if err != nil {
		return "", fmt.Errorf("reading CSS %s: %w", cssPath, err)
	}
	return string(b), nil
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

	return loadSingleCSSInDir(dir)
}

// loadSiblingCSSUnnamed discovers CSS in dir without a document basename:
//  1. style.css
//  2. styles.css
//  3. if dir contains exactly one *.css file, use that
//
// Returns "", nil if none. Same error behavior as loadSiblingCSS for I/O errors.
// Used only for content-mode discovery.
func loadSiblingCSSUnnamed(dir string) (string, error) {
	candidates := []string{
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
	return loadSingleCSSInDir(dir)
}

// loadSingleCSSInDir returns the contents of the sole *.css file in dir, or
// "", nil if zero or more than one .css files exist.
func loadSingleCSSInDir(dir string) (string, error) {
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
	if len(matches) != 1 {
		return "", nil
	}
	p := filepath.Join(dir, matches[0])
	b, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("reading CSS %s: %w", p, err)
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// Public Markdown helpers
// ---------------------------------------------------------------------------

// ConvertMarkdown converts a Markdown file to PDF without registry lookup.
// CSS is auto-discovered from sibling files unless opts.CSS or opts.CSSPath is set.
func ConvertMarkdown(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	return (&Markdown{}).Convert(ctx, inputPath, outputPath, opts)
}

// ConvertMarkdownContent converts Markdown bytes to PDF bytes using the
// Markdown converter (no registry lookup).
// opts may be nil. Prefer this over ConvertMarkdownToPDF when SourceDir or
// CSSPath is needed.
func ConvertMarkdownContent(ctx context.Context, markdown []byte, opts *Options) ([]byte, error) {
	return (&Markdown{}).ConvertContent(ctx, markdown, opts)
}

// ConvertMarkdownString is a string-input convenience for text-first callers.
// markdown is treated as UTF-8 text.
func ConvertMarkdownString(ctx context.Context, markdown string, opts *Options) ([]byte, error) {
	return ConvertMarkdownContent(ctx, []byte(markdown), opts)
}

// ConvertMarkdownToPDF converts Markdown bytes to PDF bytes using picoloom.
// css is optional; pass an empty string for no custom styling.
//
// Behavioral contract:
//   - css is applied as raw inline CSS (equivalent to Options{CSS: css})
//   - empty css means no CSSPath and no sibling discovery
//   - SourceDir is always empty (no relative asset rewrite)
//   - process CWD is not used for CSS discovery
//   - registry is not used
//
// Prefer ConvertMarkdownContent when SourceDir or CSSPath is needed.
func ConvertMarkdownToPDF(ctx context.Context, markdown []byte, css string) ([]byte, error) {
	return ConvertMarkdownContent(ctx, markdown, &Options{CSS: css})
}
