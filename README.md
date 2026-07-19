# pdf

Convert documents to PDF from the command line or from your own Go programs.

[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org/doc/go1.26)

## Features

- **Markdown → PDF** via [picoloom](https://github.com/alnah/picoloom)
- **CSS auto-discovery** — automatically picks up stylesheets next to your source files
- **Batch conversion** — convert a single file, multiple files, or auto-discover compatible files in a directory
- **Library + CLI** — use as a standalone tool or import the conversion engine into your own Go project
- **Bytes-in / bytes-out** — convert Markdown strings or `[]byte` to PDF `[]byte` without an input file path
- **Inline CSS & SourceDir** — pass stylesheets and asset bases via `Options` for both file and content APIs

## Installation

### CLI

```bash
go install pdf/cmd/pdf@latest
```

Or clone and build locally:

```bash
git clone <repo>
cd pdf
go build -o pdf ./cmd/pdf
```

### Library

```bash
go get pdf
```

## CLI Usage

### Convert a single file

```bash
pdf convert document.md
```

Output is written to `document.pdf` by default.

### Specify an output name

```bash
pdf convert document.md -o report.pdf
```

### Use a custom CSS file

```bash
pdf convert document.md --css custom.css
```

When `--css` is provided, auto-discovery is skipped and the specified stylesheet is used instead.

### Convert multiple files

```bash
pdf convert notes.md report.md proposal.md
```

### Auto-discover files

Run without arguments to scan the current directory and convert all supported files:

```bash
pdf convert
```

### Open after conversion

```bash
pdf convert document.md --open
```

## CSS Styling

### Auto-discovery

When no CSS is explicitly supplied, the converter automatically looks for a stylesheet next to your Markdown file in this order:

1. `<filename>.css` (e.g. `document.css` for `document.md`)
2. `style.css`
3. `styles.css`
4. If the directory contains exactly one `.css` file, it uses that

### Explicit CSS

You can bypass auto-discovery by providing CSS explicitly:

- **CLI:** `pdf convert document.md --css path/to/style.css`
- **Library (file path):** `&convert.Options{CSSPath: "style.css"}`
- **Library (inline content):** `&convert.Options{CSS: "body { font-family: serif; }"}`

**Precedence:** inline `CSS` (non-empty, no trimming) > `CSSPath` > sibling auto-discovery > empty.

`CSSPath` is resolved relative to the process working directory (or absolute). It is **not** rooted at `SourceDir`.

### Content-mode CSS discovery

When converting in-memory content (no input path), named `<file>.css` discovery is not available. If `SourceDir` is set and no `CSS` / `CSSPath` is provided, the converter looks under that directory for:

1. `style.css`
2. `styles.css`
3. Exactly one `.css` file in the directory

Without `SourceDir`, content conversion does not discover CSS from the working directory.

### Asset base (`SourceDir`)

`SourceDir` is the directory used to resolve relative images and other resources in the document. It is independent of CSS discovery for path-based conversion: converting `report.md` with `SourceDir: "/assets"` still discovers `report.css` next to the Markdown file.

**Security:** for untrusted Markdown, do not set `SourceDir` to broad roots such as `$HOME` or `/`. Prefer a dedicated assets directory.

## Library Usage

### Convert a file (auto-discovers CSS)

```go
package main

import (
    "context"
    "log"

    "github.com/tituscheng/pdf/pkg/convert"
)

func main() {
    ctx := context.Background()
    if err := convert.ConvertFile(ctx, "document.md", "document.pdf", nil); err != nil {
        log.Fatal(err)
    }
}
```

### Convert with a specific CSS file

```go
ctx := context.Background()
opts := &convert.Options{CSSPath: "styles.css"}
if err := convert.ConvertFile(ctx, "document.md", "document.pdf", opts); err != nil {
    log.Fatal(err)
}
```

### Convert in-memory Markdown (string or bytes)

```go
import (
    "context"
    "log"
    "os"

    "github.com/tituscheng/pdf/pkg/convert"
)

ctx := context.Background()

// String source
pdf, err := convert.ConvertMarkdownString(ctx, "# Hello World", nil)
if err != nil {
    log.Fatal(err)
}

// Bytes with Options (inline CSS + asset base for relative images)
opts := &convert.Options{
    CSS:       "body { font-family: Georgia, serif; }",
    SourceDir: "./assets",
}
pdf, err = convert.ConvertMarkdownContent(ctx, []byte("# Hello"), opts)
if err != nil {
    log.Fatal(err)
}

// Format-aware registry entry (extension required; content has no path)
pdf, err = convert.ConvertContent(ctx, ".md", []byte("# Hello"), opts)
if err != nil {
    log.Fatal(err)
}

// Legacy helper (inline CSS string only; no SourceDir)
pdf, err = convert.ConvertMarkdownToPDF(ctx, []byte("# Hello"), "")
if err != nil {
    log.Fatal(err)
}

// Write PDF bytes when you need a file
if err := os.WriteFile("out.pdf", pdf, 0o644); err != nil {
    log.Fatal(err)
}
```

Prefer `ConvertMarkdownContent` / `ConvertMarkdownString` when you need `SourceDir` or `CSSPath`. `ConvertMarkdownToPDF` remains supported as a thin wrapper.

### Use the registry

```go
ctx := context.Background()

c, err := convert.For(".md")
if err != nil {
    log.Fatal(err)
}

if err := c.Convert(ctx, "input.md", "output.pdf", nil); err != nil {
    log.Fatal(err)
}
```

### Isolated registry

For tests or concurrent use with different configurations, create your own registry instead of using the package-level default:

```go
reg := convert.NewRegistry()
reg.Register(".md", &convert.Markdown{})

opts := &convert.Options{CSSPath: "custom.css"}
if err := reg.ConvertFile(ctx, "input.md", "output.pdf", opts); err != nil {
    log.Fatal(err)
}
```

### Check for unsupported formats or content

```go
c, err := convert.For(".txt")
if errors.Is(err, convert.ErrUnsupportedFormat) {
    // no converter registered for this extension
}

_, err = convert.ConvertContent(ctx, ".myext", body, nil)
if errors.Is(err, convert.ErrContentUnsupported) {
    // converter exists but only supports file paths
}
if errors.Is(err, convert.ErrUnsupportedFormat) {
    // no converter registered
}
```

### Register a custom converter

```go
import "github.com/tituscheng/pdf/pkg/convert"

type MyConverter struct{}

func (m *MyConverter) Convert(ctx context.Context, input, output string, opts *convert.Options) error {
    // path-based conversion
    return nil
}

// Optional: implement ContentConverter for in-memory sources.
func (m *MyConverter) ConvertContent(ctx context.Context, content []byte, opts *convert.Options) ([]byte, error) {
    // content-based conversion
    return nil, nil
}

func (m *MyConverter) Name() string {
    return "My Format → PDF"
}

func init() {
    convert.Register(".myext", &MyConverter{})
}
```

## Project Structure

```
pdf/
├── cmd/
│   ├── pdf/main.go          # CLI binary entry point
│   ├── root.go              # cobra commands
│   └── convert.go           # convert command implementation
├── pkg/
│   └── convert/
│       ├── convert.go       # Converter, ContentConverter, Registry, Options
│       ├── markdown.go      # Markdown path + content converter
│       └── convert_test.go  # unit tests
├── internal/
│   └── cli/                 # CLI output helpers
└── example/
    ├── session_1.md
    └── pdf-style.css
```

## Supported Formats

| Extension | Description |
|-----------|-------------|
| `.md`     | Markdown    |
| `.markdown` | Markdown  |

## License

MIT
