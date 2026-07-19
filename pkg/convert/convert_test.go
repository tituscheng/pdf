package convert

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryRegisterAndFor(t *testing.T) {
	reg := NewRegistry()
	mock := &mockConverter{name: "Mock"}
	reg.Register(".test", mock)

	c, err := reg.For(".test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if named, ok := c.(Named); !ok || named.Name() != "Mock" {
		t.Fatalf("expected name Mock")
	}

	// lookup without leading dot
	c, err = reg.For("test")
	if err != nil {
		t.Fatalf("unexpected error for extension without dot: %v", err)
	}
	if named, ok := c.(Named); !ok || named.Name() != "Mock" {
		t.Fatalf("expected name Mock")
	}

	// unsupported extension
	_, err = reg.For(".unknown")
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestRegistrySupportedExtensions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(".a", &mockConverter{name: "A"})
	reg.Register(".b", &mockConverter{name: "B"})
	reg.Register(".c", &mockConverter{name: "C"})

	exts := reg.SupportedExtensions()
	if len(exts) != 3 {
		t.Fatalf("expected 3 extensions, got %d", len(exts))
	}
	if exts[0] != ".a" || exts[1] != ".b" || exts[2] != ".c" {
		t.Fatalf("expected sorted [.a .b .c], got %v", exts)
	}
}

func TestRegistryConvertFile(t *testing.T) {
	reg := NewRegistry()
	mock := &mockConverter{name: "Mock"}
	reg.Register(".mock", mock)

	ctx := context.Background()
	if err := reg.ConvertFile(ctx, "test.mock", "test.pdf", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.converted {
		t.Fatal("expected converter to be called")
	}

	// unsupported format
	if err := reg.ConvertFile(ctx, "test.unknown", "test.pdf", nil); err == nil {
		t.Fatal("expected error for unsupported format")
	} else if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestDefaultOutputPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/docs/report.md", "/docs/report.pdf"},
		{"report.md", "report.pdf"},
		{"/path/to/file.markdown", "/path/to/file.pdf"},
		{"file", "file.pdf"},
	}

	for _, tt := range tests {
		got := DefaultOutputPath(tt.input)
		if got != tt.expected {
			t.Errorf("DefaultOutputPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizeExt(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"md", ".md"},
		{".md", ".md"},
		{"MD", ".md"},
		{".MD", ".md"},
	}

	for _, tt := range tests {
		got := normalizeExt(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeExt(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLoadSiblingCSS(t *testing.T) {
	tmp := t.TempDir()
	mdFile := filepath.Join(tmp, "doc.md")
	if err := os.WriteFile(mdFile, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// No CSS files — should return empty string, no error.
	css, err := loadSiblingCSS(mdFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if css != "" {
		t.Fatalf("expected empty css, got %q", css)
	}

	// Single sibling CSS — should pick it up.
	styleCSS := filepath.Join(tmp, "style.css")
	if err := os.WriteFile(styleCSS, []byte("body {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	css, err = loadSiblingCSS(mdFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if css != "body {}" {
		t.Fatalf("expected css content, got %q", css)
	}

	// Named CSS takes priority over style.css.
	namedCSS := filepath.Join(tmp, "doc.css")
	if err := os.WriteFile(namedCSS, []byte("h1 {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	css, err = loadSiblingCSS(mdFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if css != "h1 {}" {
		t.Fatalf("expected named css to take priority, got %q", css)
	}
}

func TestConvertMarkdownWithOptions(t *testing.T) {
	tmp := t.TempDir()
	mdFile := filepath.Join(tmp, "doc.md")
	if err := os.WriteFile(mdFile, []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cssFile := filepath.Join(tmp, "custom.css")
	if err := os.WriteFile(cssFile, []byte("body { color: red; }"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify that a sibling style.css is ignored when CSSPath is explicit.
	styleCSS := filepath.Join(tmp, "style.css")
	if err := os.WriteFile(styleCSS, []byte("body { color: blue; }"), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmp, "out.pdf")
	ctx := context.Background()

	// Explicit CSS path via Options.
	m := &Markdown{}
	err := m.Convert(ctx, mdFile, outFile, &Options{CSSPath: cssFile})
	// picoloom may fail if no browser is available; that's ok for this test.
	// We just need to confirm no CSS-related file-not-found error.
	if err != nil && os.IsNotExist(err) {
		t.Fatalf("unexpected file not found error: %v", err)
	}
}

func TestConvertMarkdownToPDF(t *testing.T) {
	ctx := context.Background()
	pdf, err := ConvertMarkdownToPDF(ctx, []byte("# Hello"), "")
	if err != nil {
		// picoloom may fail if no browser is available.
		return
	}
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
}

func TestEffectiveSourceDir(t *testing.T) {
	tests := []struct {
		name      string
		inputPath string
		opts      *Options
		want      string
	}{
		{"nil opts path", "/docs/report.md", nil, "/docs"},
		{"empty opts path", "/docs/report.md", &Options{}, "/docs"},
		{"explicit override", "/docs/report.md", &Options{SourceDir: "/assets"}, "/assets"},
		{"content mode empty", "", nil, ""},
		{"content mode with source", "", &Options{SourceDir: "/assets"}, "/assets"},
		{"empty SourceDir falls back", "/docs/a.md", &Options{SourceDir: ""}, "/docs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveSourceDir(tt.inputPath, tt.opts)
			if got != tt.want {
				t.Fatalf("effectiveSourceDir(%q, %+v) = %q, want %q", tt.inputPath, tt.opts, got, tt.want)
			}
		})
	}
}

func TestResolveCSS_precedence(t *testing.T) {
	tmp := t.TempDir()
	mdFile := filepath.Join(tmp, "report.md")
	if err := os.WriteFile(mdFile, []byte("# Hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "report.css"), []byte("named"), 0o644); err != nil {
		t.Fatal(err)
	}
	cssPath := filepath.Join(tmp, "custom.css")
	if err := os.WriteFile(cssPath, []byte("from-path"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Inline CSS wins over CSSPath and discovery.
	css, err := resolveCSSContentForPath(mdFile, &Options{CSS: "inline", CSSPath: cssPath})
	if err != nil {
		t.Fatal(err)
	}
	if css != "inline" {
		t.Fatalf("want inline, got %q", css)
	}

	// Whitespace-only CSS still wins (no trim).
	css, err = resolveCSSContentForPath(mdFile, &Options{CSS: " ", CSSPath: cssPath})
	if err != nil {
		t.Fatal(err)
	}
	if css != " " {
		t.Fatalf("want whitespace CSS, got %q", css)
	}

	// CSSPath wins over discovery.
	css, err = resolveCSSContentForPath(mdFile, &Options{CSSPath: cssPath})
	if err != nil {
		t.Fatal(err)
	}
	if css != "from-path" {
		t.Fatalf("want from-path, got %q", css)
	}

	// Discovery when neither set.
	css, err = resolveCSSContentForPath(mdFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if css != "named" {
		t.Fatalf("want named, got %q", css)
	}

	// Content mode: empty opts → no CSS.
	css, err = resolveCSSContentForContent(nil)
	if err != nil {
		t.Fatal(err)
	}
	if css != "" {
		t.Fatalf("want empty, got %q", css)
	}

	// Content mode: inline wins.
	css, err = resolveCSSContentForContent(&Options{CSS: "c-inline", CSSPath: cssPath})
	if err != nil {
		t.Fatal(err)
	}
	if css != "c-inline" {
		t.Fatalf("want c-inline, got %q", css)
	}
}

func TestResolveCSS_inlineSkipsCSSPathStat(t *testing.T) {
	// Missing CSSPath must not error when CSS is set.
	css, err := resolveCSSContentForPath("report.md", &Options{
		CSS:     "inline",
		CSSPath: "/nonexistent/path/that/does/not/exist.css",
	})
	if err != nil {
		t.Fatalf("unexpected error when CSS is set: %v", err)
	}
	if css != "inline" {
		t.Fatalf("want inline, got %q", css)
	}

	css, err = resolveCSSContentForContent(&Options{
		CSS:     "inline",
		CSSPath: "/nonexistent/path/that/does/not/exist.css",
	})
	if err != nil {
		t.Fatalf("unexpected error when CSS is set: %v", err)
	}
	if css != "inline" {
		t.Fatalf("want inline, got %q", css)
	}
}

func TestPathConvert_basenameCSS_afterRefactor(t *testing.T) {
	tmp := t.TempDir()
	mdFile := filepath.Join(tmp, "report.md")
	if err := os.WriteFile(mdFile, []byte("# Report"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "report.css"), []byte("report-css"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "document.css"), []byte("document-css"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "style.css"), []byte("style-css"), 0o644); err != nil {
		t.Fatal(err)
	}

	css, err := resolveCSSContentForPath(mdFile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if css != "report-css" {
		t.Fatalf("expected report.css content, got %q", css)
	}
}

func TestPathConvert_SourceDirOverride_assetsOnly(t *testing.T) {
	tmp := t.TempDir()
	mdFile := filepath.Join(tmp, "report.md")
	if err := os.WriteFile(mdFile, []byte("# Report"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "report.css"), []byte("report-css"), 0o644); err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(tmp, "assets")
	if err := os.Mkdir(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "style.css"), []byte("assets-style"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := &Options{SourceDir: assets}
	if got := effectiveSourceDir(mdFile, opts); got != assets {
		t.Fatalf("effectiveSourceDir = %q, want %q", got, assets)
	}

	// CSS discovery still uses real inputPath, not SourceDir.
	css, err := resolveCSSContentForPath(mdFile, opts)
	if err != nil {
		t.Fatal(err)
	}
	if css != "report-css" {
		t.Fatalf("CSS discovery must stay on inputPath; got %q", css)
	}
}

func TestOptionsNotMutated(t *testing.T) {
	tmp := t.TempDir()
	mdFile := filepath.Join(tmp, "doc.md")
	if err := os.WriteFile(mdFile, []byte("# Hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "doc.css"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := &Options{
		CSSPath:   "original-path",
		CSS:       "original-css",
		SourceDir: "original-src",
	}
	before := *opts

	if _, err := resolveCSSContentForPath(mdFile, opts); err != nil {
		t.Fatal(err)
	}
	if *opts != before {
		t.Fatalf("resolveCSSContentForPath mutated opts: got %+v want %+v", *opts, before)
	}

	if _, err := resolveCSSContentForContent(opts); err != nil {
		t.Fatal(err)
	}
	if *opts != before {
		t.Fatalf("resolveCSSContentForContent mutated opts: got %+v want %+v", *opts, before)
	}

	_ = effectiveSourceDir(mdFile, opts)
	if *opts != before {
		t.Fatalf("effectiveSourceDir mutated opts: got %+v want %+v", *opts, before)
	}
}

func TestContentCSS_noDiscoveryWithoutSourceDir(t *testing.T) {
	// Even if CWD has style.css, content mode without SourceDir must not discover.
	// We only assert the resolver contract (empty SourceDir → no discovery call).
	css, err := resolveCSSContentForContent(&Options{})
	if err != nil {
		t.Fatal(err)
	}
	if css != "" {
		t.Fatalf("want empty CSS without SourceDir, got %q", css)
	}
}

func TestContentCSS_unnamedDiscovery(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "style.css"), []byte("style-body"), 0o644); err != nil {
		t.Fatal(err)
	}
	css, err := resolveCSSContentForContent(&Options{SourceDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if css != "style-body" {
		t.Fatalf("want style-body, got %q", css)
	}

	// Sole non-standard name.
	tmp2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp2, "pdf-style.css"), []byte("sole"), 0o644); err != nil {
		t.Fatal(err)
	}
	css, err = resolveCSSContentForContent(&Options{SourceDir: tmp2})
	if err != nil {
		t.Fatal(err)
	}
	if css != "sole" {
		t.Fatalf("want sole css file, got %q", css)
	}

	// Two non-named files → no single match, and no style.css.
	tmp3 := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp3, "a.css"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp3, "b.css"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	css, err = resolveCSSContentForContent(&Options{SourceDir: tmp3})
	if err != nil {
		t.Fatal(err)
	}
	if css != "" {
		t.Fatalf("want empty when multiple unnamed css, got %q", css)
	}
}

func TestContentCSS_noSyntheticDocumentName(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "document.css"), []byte("document-css"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "style.css"), []byte("style-css"), 0o644); err != nil {
		t.Fatal(err)
	}

	css, err := resolveCSSContentForContent(&Options{SourceDir: tmp})
	if err != nil {
		t.Fatal(err)
	}
	if css != "style-css" {
		t.Fatalf("content mode must prefer style.css over document.css, got %q", css)
	}
}

func TestRegistry_ConvertContent(t *testing.T) {
	reg := NewRegistry()
	mock := &mockContentConverter{name: "ContentMock"}
	reg.Register(".mock", mock)

	ctx := context.Background()
	pdf, err := reg.ConvertContent(ctx, ".mock", []byte("hello"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(pdf) != "pdf:hello" {
		t.Fatalf("got %q", pdf)
	}
	if !mock.converted {
		t.Fatal("expected ConvertContent to be called")
	}
}

func TestRegistry_ConvertContent_unsupportedFormat(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.ConvertContent(context.Background(), ".unknown", []byte("x"), nil)
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestRegistry_ConvertContent_pathOnlyConverter(t *testing.T) {
	reg := NewRegistry()
	reg.Register(".pathonly", &mockConverter{name: "PathOnly"})

	_, err := reg.ConvertContent(context.Background(), ".pathonly", []byte("x"), nil)
	if !errors.Is(err, ErrContentUnsupported) {
		t.Fatalf("expected ErrContentUnsupported, got %v", err)
	}
}

func TestConvertMarkdownToPDF_contract(t *testing.T) {
	// Empty CSS: no discovery even if we cannot easily pollute CWD;
	// contract is exercised via resolve path used by ConvertMarkdownContent.
	css, err := resolveCSSContentForContent(&Options{CSS: ""})
	if err != nil {
		t.Fatal(err)
	}
	if css != "" {
		t.Fatalf("empty CSS must not discover, got %q", css)
	}

	// Non-empty css maps to Options.CSS.
	css, err = resolveCSSContentForContent(&Options{CSS: "body{}"})
	if err != nil {
		t.Fatal(err)
	}
	if css != "body{}" {
		t.Fatalf("got %q", css)
	}

	// SourceDir effectively empty for ConvertMarkdownToPDF path.
	if got := effectiveSourceDir("", &Options{CSS: "x"}); got != "" {
		t.Fatalf("SourceDir should be empty for ConvertMarkdownToPDF opts, got %q", got)
	}

	// Integration: may skip if no browser.
	ctx := context.Background()
	pdf, err := ConvertMarkdownToPDF(ctx, []byte("# Contract"), "h1{color:red}")
	if err != nil {
		return
	}
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF")
	}
}

func TestConvertMarkdownString(t *testing.T) {
	ctx := context.Background()
	// Without browser, both should fail the same way or both succeed.
	a, errA := ConvertMarkdownString(ctx, "# Hello", nil)
	b, errB := ConvertMarkdownContent(ctx, []byte("# Hello"), nil)
	if (errA == nil) != (errB == nil) {
		t.Fatalf("string and bytes helpers diverged: errA=%v errB=%v", errA, errB)
	}
	if errA != nil {
		return
	}
	if len(a) == 0 || len(b) == 0 {
		t.Fatal("expected non-empty PDF from both helpers")
	}
}

func TestMarkdown_ConvertContent_basic(t *testing.T) {
	ctx := context.Background()
	pdf, err := (&Markdown{}).ConvertContent(ctx, []byte("# Hello"), nil)
	if err != nil {
		// picoloom may fail if no browser is available.
		return
	}
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
}

type mockConverter struct {
	name      string
	converted bool
}

func (m *mockConverter) Convert(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	m.converted = true
	return nil
}

func (m *mockConverter) Name() string {
	return m.name
}

type mockContentConverter struct {
	name      string
	converted bool
}

func (m *mockContentConverter) Convert(ctx context.Context, inputPath, outputPath string, opts *Options) error {
	return nil
}

func (m *mockContentConverter) ConvertContent(ctx context.Context, content []byte, opts *Options) ([]byte, error) {
	m.converted = true
	return []byte("pdf:" + string(content)), nil
}

func (m *mockContentConverter) Name() string {
	return m.name
}
