package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestMainGeneratesTemplateIcon(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	workDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(workDir, "assets"), 0o700); err != nil {
		t.Fatalf("creating assets directory: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("changing working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	main()

	iconPath := filepath.Join(workDir, "assets", "iconTemplate.png")
	file, err := os.Open(iconPath)
	if err != nil {
		t.Fatalf("opening generated icon: %v", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decoding generated icon: %v", err)
	}

	if got := img.Bounds().Dx(); got != 22 {
		t.Fatalf("expected width 22, got %d", got)
	}
	if got := img.Bounds().Dy(); got != 22 {
		t.Fatalf("expected height 22, got %d", got)
	}

	centerR, centerG, centerB, centerA := img.At(11, 11).RGBA()
	if centerA == 0 {
		t.Fatal("expected center pixel to be opaque")
	}
	if centerR != 0 || centerG != 0 || centerB != 0 {
		t.Fatalf("expected center pixel to be black, got r=%d g=%d b=%d", centerR, centerG, centerB)
	}

	_, _, _, cornerA := img.At(0, 0).RGBA()
	if cornerA != 0 {
		t.Fatalf("expected corner pixel to be transparent, alpha=%d", cornerA)
	}
}
