// Package create implements the agentws create subcommand.
//
// Create scaffolds an agent workspace by merging files from a source
// configuration directory into a target directory. The merge is non-
// destructive by default: existing files in the target are preserved.
// Pass Options.Force to overwrite conflicting files.
package create

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrSourceMissing indicates the source config directory does not exist.
var ErrSourceMissing = errors.New("source config directory not found")

// Subdirs lists the standard workspace subdirectories to scaffold.
// A missing subdirectory in the source is treated as optional.
var Subdirs = []string{"skills", "instructions", "agents", "docs"}

// ConfigFile is the name of the configuration file in the workspace root.
const ConfigFile = "agentws.toml"

// Options configures a Create run.
type Options struct {
	Source string    // source config directory (required)
	Target string    // target directory, typically the cwd (required)
	Force  bool      // overwrite conflicting files in the target
	Stdout io.Writer // status output for created files; defaults to os.Stdout
	Stderr io.Writer // status output for skipped files; defaults to os.Stderr
}

// Result summarizes the actions taken by Create.
type Result struct {
	Copied  []string // paths relative to Target that were written
	Skipped []string // paths relative to Target that were preserved
}

// Create scaffolds the agent workspace from Source into Target.
//
// It walks each known Subdir in Source and merges its contents into the
// matching directory under Target. Files already present in Target are
// preserved unless opts.Force is true. The agentws.toml at the source
// root is copied to the target root if present.
//
// Missing optional subdirectories in Source are skipped silently.
// A missing Source returns an error wrapping ErrSourceMissing.
func Create(opts Options) (Result, error) {
	if opts.Source == "" || opts.Target == "" {
		return Result{}, errors.New("source and target are required")
	}
	srcInfo, err := os.Stat(opts.Source)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Result{}, fmt.Errorf("%w: %s", ErrSourceMissing, opts.Source)
		}
		return Result{}, err
	}
	if !srcInfo.IsDir() {
		return Result{}, fmt.Errorf("source is not a directory: %s", opts.Source)
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	var res Result

	for _, sub := range Subdirs {
		subSrc := filepath.Join(opts.Source, sub)
		if _, err := os.Stat(subSrc); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return res, err
		}
		if err := mergeDir(subSrc, filepath.Join(opts.Target, sub), sub, opts.Force, &res); err != nil {
			return res, err
		}
	}

	cfgSrc := filepath.Join(opts.Source, ConfigFile)
	if _, err := os.Stat(cfgSrc); err == nil {
		action, err := copyOrSkip(cfgSrc, filepath.Join(opts.Target, ConfigFile), opts.Force)
		if err != nil {
			return res, err
		}
		recordAction(&res, ConfigFile, action)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return res, err
	}

	for _, p := range res.Copied {
		fmt.Fprintf(stdout, "created %s\n", p)
	}
	for _, p := range res.Skipped {
		fmt.Fprintf(stderr, "skipped %s (exists; use --force to overwrite)\n", p)
	}
	return res, nil
}

// mergeDir recursively copies regular files from srcDir into dstDir.
// rel is the path of dstDir relative to the target root, used for
// status reporting.
func mergeDir(srcDir, dstDir, rel string, force bool, res *Result) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())
		entryRel := filepath.Join(rel, e.Name())
		if e.IsDir() {
			if err := mergeDir(src, dst, entryRel, force, res); err != nil {
				return err
			}
			continue
		}
		action, err := copyOrSkip(src, dst, force)
		if err != nil {
			return err
		}
		recordAction(res, entryRel, action)
	}
	return nil
}

// copyOrSkip copies src to dst unless dst already exists. When force
// is true a pre-existing dst is removed and overwritten. The returned
// string is "copied" or "skipped".
func copyOrSkip(src, dst string, force bool) (string, error) {
	if _, err := os.Lstat(dst); err == nil {
		if !force {
			return "skipped", nil
		}
		if err := os.RemoveAll(dst); err != nil {
			return "", err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err := copyFile(src, dst); err != nil {
		return "", err
	}
	return "copied", nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func recordAction(res *Result, rel, action string) {
	switch action {
	case "copied":
		res.Copied = append(res.Copied, rel)
	case "skipped":
		res.Skipped = append(res.Skipped, rel)
	}
}
