// Package cli provides user-friendly, colourised terminal messages.
package cli

import (
	"os"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen, color.Bold)
	red    = color.New(color.FgRed, color.Bold)
	yellow = color.New(color.FgYellow, color.Bold)
	cyan   = color.New(color.FgCyan, color.Bold)
)

// Success prints a success message.
func Success(format string, a ...any) {
	_, _ = green.Fprintf(os.Stdout, "✓ "+format+"\n", a...)
}

// Error prints an error message.
func Error(format string, a ...any) {
	_, _ = red.Fprintf(os.Stderr, "✗ "+format+"\n", a...)
}

// Warn prints a warning message.
func Warn(format string, a ...any) {
	_, _ = yellow.Fprintf(os.Stderr, "! "+format+"\n", a...)
}

// Info prints an informational message.
func Info(format string, a ...any) {
	_, _ = cyan.Fprintf(os.Stdout, "→ "+format+"\n", a...)
}

// Fatal prints an error message and exits the program with status 1.
func Fatal(format string, a ...any) {
	Error(format, a...)
	os.Exit(1)
}

// Result summarises the conversion outcome for a single file.
func Result(input, output string, err error) {
	if err != nil {
		Error("%s  →  %s", input, err)
		return
	}
	Success("%s  →  %s", input, output)
}
