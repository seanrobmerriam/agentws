// Command agentws scaffolds an agent workspace from a source config
// directory (default: $AGENTWS_HOME or ~/.config/agentws) into the
// current working directory. Use `agentws create` to merge, or
// `agentws create --force` to overwrite conflicting files.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"agentws/internal/create"
)

const usage = `agentws — scaffold an agent workspace

Usage:
  agentws create [--force] [--source DIR]

Commands:
  create   Scaffold an agent workspace into the current directory

Flags:
  --force         Overwrite existing files in the target
  --source DIR    Source config directory
                  (default: $AGENTWS_HOME or ~/.config/agentws)
  -h, --help      Show this help
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "create":
		os.Exit(runCreate(os.Args[2:]))
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func runCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		force  bool
		source string
	)
	fs.BoolVar(&force, "force", false, "overwrite existing files in the target")
	fs.StringVar(&source, "source", "", "source config directory (default: $AGENTWS_HOME or ~/.config/agentws)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if source == "" {
		source = os.Getenv("AGENTWS_HOME")
	}
	if source == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "agentws: cannot determine home directory: %v\n", err)
			return 1
		}
		source = filepath.Join(home, ".config", "agentws")
	}

	target, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentws: cannot determine working directory: %v\n", err)
		return 1
	}

	_, err = create.Create(create.Options{
		Source: source,
		Target: target,
		Force:  force,
	})
	if err != nil {
		if errors.Is(err, create.ErrSourceMissing) {
			fmt.Fprintf(os.Stderr, "agentws: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "agentws: %v\n", err)
		return 1
	}
	return 0
}
