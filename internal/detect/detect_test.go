package detect

import (
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/bjro/agentbox/internal/stack"
)

func TestDetect_SingleStack_Go(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Go}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_SingleStack_Node(t *testing.T) {
	fsys := fstest.MapFS{
		"package.json": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Node}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_SingleStack_Node_TsconfigOnly(t *testing.T) {
	fsys := fstest.MapFS{
		"tsconfig.json": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Node}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_SingleStack_Node_BothMarkers(t *testing.T) {
	fsys := fstest.MapFS{
		"package.json":  &fstest.MapFile{},
		"tsconfig.json": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Node}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_SingleStack_Python(t *testing.T) {
	markers := []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"}

	for _, marker := range markers {
		t.Run(marker, func(t *testing.T) {
			fsys := fstest.MapFS{
				marker: &fstest.MapFile{},
			}

			got, err := detect(fsys)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := []stack.StackID{stack.Python}
			if !slices.Equal(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestDetect_SingleStack_Rust(t *testing.T) {
	fsys := fstest.MapFS{
		"Cargo.toml": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Rust}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_SingleStack_Ruby_Gemfile(t *testing.T) {
	fsys := fstest.MapFS{
		"Gemfile": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Ruby}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_SingleStack_Ruby_Gemspec(t *testing.T) {
	fsys := fstest.MapFS{
		"foo.gemspec": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Ruby}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_MultiStack(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod":       &fstest.MapFile{},
		"package.json": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Go, stack.Node}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_AllStacks(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod":           &fstest.MapFile{},
		"package.json":     &fstest.MapFile{},
		"requirements.txt": &fstest.MapFile{},
		"Cargo.toml":       &fstest.MapFile{},
		"Gemfile":          &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Ruby, stack.Rust}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_NoStacks(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md": &fstest.MapFile{},
		"LICENSE":   &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestDetect_MarkerInSubdir(t *testing.T) {
	fsys := fstest.MapFS{
		"subproject/go.mod": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Go}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_MarkerTwoLevelsDeep_NotDetected(t *testing.T) {
	fsys := fstest.MapFS{
		"deep/nested/go.mod": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice for two-levels-deep marker, got %v", got)
	}
}

func TestDetect_SkipsVendorDir(t *testing.T) {
	fsys := fstest.MapFS{
		"vendor/Cargo.toml": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice when marker is in vendor/, got %v", got)
	}
}

func TestDetect_SkipsNodeModules(t *testing.T) {
	fsys := fstest.MapFS{
		"node_modules/package.json": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice when marker is in node_modules/, got %v", got)
	}
}

func TestDetect_SkipsGitDir(t *testing.T) {
	fsys := fstest.MapFS{
		".git/go.mod": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice when marker is in .git/, got %v", got)
	}
}

func TestDetect_GemspecInSubdir(t *testing.T) {
	fsys := fstest.MapFS{
		"mylib/foo.gemspec": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Ruby}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDetect_GemspecInSkipDir_NotDetected(t *testing.T) {
	fsys := fstest.MapFS{
		"vendor/foo.gemspec": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty slice when gemspec is in vendor/, got %v", got)
	}
}

func TestDetect_ResultIsSorted(t *testing.T) {
	fsys := fstest.MapFS{
		"Cargo.toml":       &fstest.MapFile{},
		"go.mod":           &fstest.MapFile{},
		"requirements.txt": &fstest.MapFile{},
		"package.json":     &fstest.MapFile{},
		"Gemfile":          &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	strs := make([]string, len(got))
	for i, id := range got {
		strs[i] = string(id)
	}
	if !slices.IsSorted(strs) {
		t.Errorf("result is not sorted: got %v", got)
	}
}

func TestDetect_EmptyResult_IsNonNil(t *testing.T) {
	fsys := fstest.MapFS{}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestDetect_NoDuplicates(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod":            &fstest.MapFile{},
		"subproject/go.mod": &fstest.MapFile{},
	}

	got, err := detect(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []stack.StackID{stack.Go}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v (expected no duplicates)", got, want)
	}
}

func TestDetect_PublicAPI_InvalidDir(t *testing.T) {
	_, err := Detect("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}

func TestDetect_PublicAPI_NotADirectory(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "notadir")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Detect(f.Name())
	if err == nil {
		t.Error("expected error for file path, got nil")
	}
}
