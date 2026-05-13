# pdf

Convert documents to PDF from the command line or from your own Go programs.

[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org/doc/go1.26)

## Features

- **Markdown → PDF** via [picoloom](https://github.com/alnah/picoloom)
- **CSS auto-discovery** — automatically picks up stylesheets next to your source files
- **Batch conversion** — convert a single file, multiple files, or auto-discover compatible files in a directory
- **Library + CLI** — use as a standalone tool or import the conversion engine into your own Go project
- **Bytes-in / bytes-out** — convert Markdown `[]byte` to PDF `[]byte` without touching the filesystem

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

You can bypass auto-discovery by providing a CSS file explicitly:

- **CLI:** `pdf convert document.md --css path/to/style.css`
- **Library:** pass `&convert.Options{CSSPath: "style.css"}`

When a CSS path is provided, auto-discovery is skipped entirely.

## Library Usage

### Convert a file (auto-discovers CSS)

```go
package main

import (
    "context"
    "log"

    "pdf/pkg/convert"
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

### Convert Markdown bytes to PDF bytes

```go
ctx := context.Background()
markdown := []byte("# Hello World")
pdfBytes, err := convert.ConvertMarkdownToPDF(ctx, markdown, "")
if err != nil {
    log.Fatal(err)
}
// pdfBytes is a PDF document ready to write or serve
```

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

### Check for unsupported formats

```go
c, err := convert.For(".txt")
if errors.Is(err, convert.ErrUnsupportedFormat) {
    // handle unsupported format
}
```

### Register a custom converter

```go
import "pdf/pkg/convert"

type MyConverter struct{}

func (m *MyConverter) Convert(ctx context.Context, input, output string, opts *convert.Options) error {
    // your conversion logic
    return nil
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
│       ├── convert.go       # Converter interface, Registry, Options
│       ├── markdown.go      # Markdown converter
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
