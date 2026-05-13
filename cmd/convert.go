package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/tituscheng/pdf/internal/cli"
	"github.com/tituscheng/pdf/pkg/convert"

	"github.com/spf13/cobra"
)

var outputFlag string
var openFlag bool
var cssFlag string

var convertCmd = &cobra.Command{
	Use:   "convert [file ...]",
	Short: "Convert files to PDF",
	Long: `Convert one or more files to PDF format.

If no files are provided, the current directory is scanned for
compatible files and they are converted automatically.

Supported formats:
  • Markdown (.md, .markdown)

Examples:
  pdf convert document.md
  pdf convert notes.md report.md
  pdf convert proposal.md -o final.pdf
  pdf convert                # auto-discover compatible files`,
	Args: cobra.ArbitraryArgs,
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output filename (only valid with a single input file)")
	convertCmd.Flags().BoolVar(&openFlag, "open", false, "Open the PDF file after conversion")
	convertCmd.Flags().StringVar(&cssFlag, "css", "", "Path to a custom CSS file (disables auto-discovery)")
	rootCmd.AddCommand(convertCmd)
}

func runConvert(cmd *cobra.Command, args []string) error {
	if outputFlag != "" && len(args) > 1 {
		return errors.New("--output can only be used with a single input file")
	}

	var files []string
	var err error

	if len(args) == 0 {
		files, err = discoverFiles(".")
		if err != nil {
			return err
		}
	} else {
		files, err = resolveInputs(args)
		if err != nil {
			return err
		}
	}

	// When a single file is passed without --output, default to <name>.pdf.
	defaultOutput := outputFlag
	if defaultOutput == "" && len(files) == 1 {
		defaultOutput = convert.DefaultOutputPath(files[0])
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opts *convert.Options
	if cssFlag != "" {
		opts = &convert.Options{CSSPath: cssFlag}
	}

	var hadError bool
	for i, file := range files {
		outPath := defaultOutput
		if len(files) > 1 {
			outPath = convert.DefaultOutputPath(file)
		}

		cli.Info("[%d/%d] Converting %s …", i+1, len(files), filepath.Base(file))

		ext := filepath.Ext(file)
		c, convErr := convert.For(ext)
		if convErr != nil {
			cli.Result(file, "", fmt.Errorf("unsupported format %q", ext))
			hadError = true
			continue
		}

		if err := c.Convert(ctx, file, outPath, opts); err != nil {
			cli.Result(file, "", err)
			hadError = true
			continue
		}

		cli.Result(file, outPath, nil)

		if openFlag {
			if err := openFile(outPath); err != nil {
				cli.Warn("Failed to open: %v", err)
			}
		}
	}

	if hadError {
		return errors.New("one or more conversions failed")
	}
	return nil
}

// resolveInputs expands directories (not yet supported) and validates that
// every argument exists and is a regular file.
func resolveInputs(args []string) ([]string, error) {
	files := make([]string, 0, len(args))
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("file not found: %s", arg)
			}
			return nil, fmt.Errorf("cannot access %s: %w", arg, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("directories are not supported yet: %s", arg)
		}
		files = append(files, arg)
	}
	return files, nil
}

// discoverFiles scans dir for files with supported extensions and returns them
// sorted alphabetically.
func discoverFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory %q: %w", dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if _, err := convert.For(ext); err == nil {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)

	if len(files) == 0 {
		exts := convert.SupportedExtensions()
		return nil, fmt.Errorf("no compatible files found in %q (looking for: %v)", dir, exts)
	}

	return files, nil
}

func openFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
