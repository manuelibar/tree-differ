package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mibar/tree-differ/pkg/differ"
)

const usage = `differ — structured JSON diffing

Usage:
  differ [flags] [left.json] [right.json]

Positional arguments:
  left.json   Left (baseline) document
  right.json  Right (target) document

Flags:
  -left string        Left document file path
  -right string       Right document file path
  -left-input string  Left document as inline JSON
  -right-input string Right document as inline JSON
  -format string      Output format: delta (default), patch, merge, stat, paths
  -pretty             Pretty-print JSON output
  -only string        Comma-separated JSONPath prefixes to include
  -ignore string      Comma-separated JSONPath prefixes to exclude
  -max-depth int      Maximum traversal depth (0=unlimited, default: 1000)
  -output string      Write output to file instead of stdout

Exit codes:
  0  Documents are equal
  1  Documents differ
  2  Error (invalid input, I/O failure)`

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("differ", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	leftFile := fs.String("left", "", "left document file path")
	rightFile := fs.String("right", "", "right document file path")
	leftInput := fs.String("left-input", "", "left document as inline JSON")
	rightInput := fs.String("right-input", "", "right document as inline JSON")
	format := fs.String("format", "", "output format: delta, patch, merge, stat, paths")
	pretty := fs.Bool("pretty", false, "pretty-print JSON output")
	only := fs.String("only", "", "comma-separated JSONPath prefixes to include")
	ignore := fs.String("ignore", "", "comma-separated JSONPath prefixes to exclude")
	maxDepth := fs.Int("max-depth", 0, fmt.Sprintf("maximum depth (0=unlimited, default: %d)", differ.DefaultMaxDepth))
	output := fs.String("output", "", "write output to file")

	fs.Usage = func() { fmt.Fprintln(os.Stderr, usage) }

	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}

	// Resolve positional args into left/right files
	args := fs.Args()
	if len(args) >= 2 && *leftFile == "" && *leftInput == "" {
		*leftFile = args[0]
		*rightFile = args[1]
	} else if len(args) == 1 && *leftFile == "" && *leftInput == "" {
		*leftFile = args[0]
	}

	// Read left document
	leftData, err := resolveInput(*leftFile, *leftInput, "left")
	if err != nil {
		fmt.Fprintf(os.Stderr, "differ: %v\n", err)
		return 2
	}

	// Read right document
	rightData, err := resolveInput(*rightFile, *rightInput, "right")
	if err != nil {
		fmt.Fprintf(os.Stderr, "differ: %v\n", err)
		return 2
	}

	// Parse format
	f, err := differ.ParseFormat(*format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "differ: %v\n", err)
		return 2
	}

	// Build options
	var opts []differ.Option
	opts = append(opts, differ.WithFormat(f))
	opts = append(opts, differ.WithPretty(*pretty))

	if *only != "" {
		opts = append(opts, differ.WithOnly(splitComma(*only)...))
	}
	if *ignore != "" {
		opts = append(opts, differ.WithIgnore(splitComma(*ignore)...))
	}
	if *maxDepth != 0 {
		opts = append(opts, differ.WithLimits(differ.Limits{MaxDepth: differ.Ptr(*maxDepth)}))
	}

	// Run diff
	result, err := differ.Diff(leftData, rightData, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "differ: %v\n", err)
		return 2
	}

	// Format output
	out, err := differ.FormatResult(result, f, *pretty)
	if err != nil {
		fmt.Fprintf(os.Stderr, "differ: %v\n", err)
		return 2
	}

	// Write output
	if *output != "" {
		if err := os.WriteFile(*output, append(out, '\n'), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "differ: write %s: %v\n", *output, err)
			return 2
		}
	} else {
		os.Stdout.Write(out)
		os.Stdout.Write([]byte{'\n'})
	}

	if result.Equal {
		return 0
	}
	return 1
}

// resolveInput reads a document from file, inline string, or stdin.
func resolveInput(file, inline, label string) ([]byte, error) {
	switch {
	case file != "" && file != "-":
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", label, err)
		}
		return data, nil
	case inline != "":
		return []byte(inline), nil
	default:
		// Check if stdin has data (only for left document)
		if label == "left" {
			stat, _ := os.Stdin.Stat()
			if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return nil, fmt.Errorf("read stdin: %w", err)
				}
				return data, nil
			}
		}
		return nil, fmt.Errorf("%s document not provided", label)
	}
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
