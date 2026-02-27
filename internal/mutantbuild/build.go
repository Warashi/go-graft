package mutantbuild

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/Warashi/go-graft/internal/model"
)

type Builder struct {
	BaseTempDir string
}

type Input struct {
	ID         string
	RuleName   string
	Point      model.MutationPoint
	Mutated    *ast.File
	Fset       *token.FileSet
	BaseTempID string
}

func (b Builder) Build(in Input) (*model.Mutant, error) {
	if in.Mutated == nil {
		return nil, fmt.Errorf("mutantbuild: mutated file must not be nil")
	}
	if in.Fset == nil {
		return nil, fmt.Errorf("mutantbuild: fset must not be nil")
	}
	orig := in.Point.CompiledGoFile
	if orig == "" {
		orig = in.Point.FilePath
	}
	if orig == "" {
		return nil, fmt.Errorf("mutantbuild: original file path is empty")
	}

	prefix := "graft-mutant-"
	if in.BaseTempID != "" {
		prefix = "graft-" + sanitize(in.BaseTempID) + "-"
	}
	tempDir, err := os.MkdirTemp(b.BaseTempDir, prefix)
	if err != nil {
		return nil, fmt.Errorf("mutantbuild: create temp dir: %w", err)
	}

	overlayDir := filepath.Join(tempDir, "overlay")
	tmpDir := filepath.Join(tempDir, "tmp")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		return nil, fmt.Errorf("mutantbuild: create overlay dir: %w", err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("mutantbuild: create tmp dir: %w", err)
	}

	mutantFile := filepath.Join(overlayDir, filepath.Base(orig))
	var buf bytes.Buffer
	if err := format.Node(&buf, in.Fset, in.Mutated); err != nil {
		return nil, fmt.Errorf("mutantbuild: format mutated file: %w", err)
	}
	if err := os.WriteFile(mutantFile, buf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("mutantbuild: write mutated file: %w", err)
	}

	overlayPath := filepath.Join(tempDir, "overlay.json")
	replace := map[string]string{
		orig: mutantFile,
	}
	if err := writeOverlay(overlayPath, replace); err != nil {
		return nil, err
	}

	return &model.Mutant{
		ID:            in.ID,
		RuleName:      in.RuleName,
		Point:         in.Point,
		TempDir:       tempDir,
		OverlayPath:   overlayPath,
		OverlayTmpDir: tmpDir,
		MutantFile:    mutantFile,
		ReplaceMap:    replace,
	}, nil
}

func writeOverlay(path string, replace map[string]string) error {
	payload := struct {
		Replace map[string]string `json:"Replace"`
	}{
		Replace: replace,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("mutantbuild: marshal overlay: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("mutantbuild: write overlay file: %w", err)
	}
	return nil
}

func sanitize(v string) string {
	v = strings.ToLower(v)
	replacer := strings.NewReplacer("/", "-", " ", "-", "_", "-", ":", "-", "@", "-", ".", "-")
	v = replacer.Replace(v)
	if v == "" {
		return "mutant"
	}
	return v
}
