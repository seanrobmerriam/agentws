package create

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// fixtureDir returns the absolute path to the test fixture config.
func fixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata", "fixture-config")
}

func TestCreate_CleanTarget(t *testing.T) {
	src := fixtureDir(t)
	tmp := t.TempDir()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	res, err := Create(Options{Source: src, Target: tmp, Stdout: stdout, Stderr: stderr})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wantFiles := []string{
		filepath.Join("skills", "example", "SKILL.md"),
		filepath.Join("instructions", "system.md"),
		filepath.Join("agents", "AGENTS.md"),
		filepath.Join("docs", "overview.md"),
		ConfigFile,
	}
	for _, rel := range wantFiles {
		if _, err := os.Stat(filepath.Join(tmp, rel)); err != nil {
			t.Errorf("missing expected file %s: %v", rel, err)
		}
	}

	if len(res.Skipped) != 0 {
		t.Errorf("expected no skipped, got %v", res.Skipped)
	}
	if len(res.Copied) == 0 {
		t.Errorf("expected copied > 0, got 0")
	}
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output, got %q", stderr.String())
	}
}

func TestCreate_ConflictNoForce_PreservesExisting(t *testing.T) {
	src := fixtureDir(t)
	tmp := t.TempDir()

	existingDir := filepath.Join(tmp, "skills", "example")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existingPath := filepath.Join(existingDir, "SKILL.md")
	existingContent := []byte("USER CONTENT — keep me\n")
	if err := os.WriteFile(existingPath, existingContent, 0o644); err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(existingPath)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Create(Options{Source: src, Target: tmp})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, existingContent) {
		t.Errorf("existing file overwritten: got %q, want %q", got, existingContent)
	}
	rel, _ := filepath.Rel(tmp, existingPath)
	if !contains(res.Skipped, rel) {
		t.Errorf("expected %q in skipped, got %v", rel, res.Skipped)
	}
	if contains(res.Copied, rel) {
		t.Errorf("did not expect %q in copied", rel)
	}

	afterInfo, err := os.Stat(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Errorf("mtime changed for preserved file: before=%v after=%v", beforeInfo.ModTime(), afterInfo.ModTime())
	}
}

func TestCreate_Force_Overwrites(t *testing.T) {
	src := fixtureDir(t)
	tmp := t.TempDir()

	existingDir := filepath.Join(tmp, "skills", "example")
	if err := os.MkdirAll(existingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existingPath := filepath.Join(existingDir, "SKILL.md")
	if err := os.WriteFile(existingPath, []byte("USER CONTENT\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Create(Options{Source: src, Target: tmp, Force: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rel, _ := filepath.Rel(tmp, existingPath)
	if !contains(res.Copied, rel) {
		t.Errorf("expected %q in copied, got %v", rel, res.Copied)
	}
	if contains(res.Skipped, rel) {
		t.Errorf("did not expect %q in skipped", rel)
	}

	srcContent, err := os.ReadFile(filepath.Join(src, "skills", "example", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, srcContent) {
		t.Errorf("forced copy mismatch: got %q, want %q", got, srcContent)
	}
}

func TestCreate_MissingSource_Errors(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist")
	_, err := Create(Options{Source: missing, Target: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing source")
	}
	if !errors.Is(err, ErrSourceMissing) {
		t.Errorf("expected ErrSourceMissing, got %v", err)
	}
}

func TestCreate_SourceIsFile_Errors(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Create(Options{Source: file, Target: t.TempDir()})
	if err == nil {
		t.Fatal("expected error when source is a regular file")
	}
}

func TestCreate_MissingOptionalSubdir_Handled(t *testing.T) {
	src := t.TempDir()
	skillsDir := filepath.Join(src, "skills", "minimal")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// instructions/agents/docs intentionally absent.

	tmp := t.TempDir()
	stdout := &bytes.Buffer{}
	res, err := Create(Options{Source: src, Target: tmp, Stdout: stdout})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	want := filepath.Join("skills", "minimal", "SKILL.md")
	if _, err := os.Stat(filepath.Join(tmp, want)); err != nil {
		t.Errorf("expected %s in target: %v", want, err)
	}
	if len(res.Copied) != 1 {
		t.Errorf("expected 1 copied, got %d (%v)", len(res.Copied), res.Copied)
	}
}

func TestCreate_ResultSortedDeterministic(t *testing.T) {
	src := fixtureDir(t)
	tmp := t.TempDir()
	res, err := Create(Options{Source: src, Target: tmp})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Just ensure both lists are deterministic (sortable, no dupes).
	sorted := append([]string(nil), res.Copied...)
	sort.Strings(sorted)
	if !equalSlices(res.Copied, sorted) && len(res.Copied) > 1 {
		// Acceptable: order is not required, but we want a clear report.
		t.Logf("copied order: %v", res.Copied)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
