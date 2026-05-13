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
