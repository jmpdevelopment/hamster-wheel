package cv

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractProfileText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cv.txt")
	if err := os.WriteFile(path, []byte("Senior Go backend engineer\nDistributed systems"), 0o600); err != nil {
		t.Fatalf("writing text cv: %v", err)
	}

	profile, err := ExtractProfile(path)
	if err != nil {
		t.Fatalf("extracting text profile: %v", err)
	}
	if !strings.Contains(profile, "Senior Go backend engineer") {
		t.Fatalf("expected extracted text profile to include content, got %q", profile)
	}
}

func TestExtractProfilePDF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cv.pdf")
	raw := buildMinimalPDF("Go backend engineer with distributed systems experience")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("writing pdf cv: %v", err)
	}

	profile, err := ExtractProfile(path)
	if err != nil {
		t.Fatalf("extracting pdf profile: %v", err)
	}
	if !strings.Contains(profile, "Go backend engineer") {
		t.Fatalf("expected extracted pdf profile to include text, got %q", profile)
	}
}

func TestExtractProfileRejectsUnsupportedZipDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cv.docx")
	raw := append([]byte("PK\x03\x04"), []byte("fake-word-document")...)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("writing fake docx: %v", err)
	}

	_, err := ExtractProfile(path)
	if err == nil {
		t.Fatal("expected unsupported-format error for zip document")
	}
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestExtractProfileRejectsDirectories(t *testing.T) {
	_, err := ExtractProfile(t.TempDir())
	if err == nil {
		t.Fatal("expected error for directory path")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Fatalf("expected directory error, got %v", err)
	}
}

func buildMinimalPDF(text string) []byte {
	escaped := escapePDFText(text)
	stream := fmt.Sprintf("BT\n/F1 12 Tf\n72 720 Td\n(%s) Tj\nET\n", escaped)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)

	return out.Bytes()
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}
