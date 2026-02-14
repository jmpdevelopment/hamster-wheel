package cv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

const (
	maxTextFileBytes        = 512 * 1024
	maxExtractedPDFTextByte = 2 * 1024 * 1024
	maxProfileRunes         = 2200
)

var (
	ErrUnsupportedFormat = errors.New("unsupported cv format")
	ErrNoUsableText      = errors.New("cv file contains no usable text")
)

var (
	pdfHeader  = []byte("%PDF-")
	zipHeader  = []byte("PK\x03\x04")
	docHeader  = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	bomUTF8    = []byte{0xEF, 0xBB, 0xBF}
	maxSigRead = 16
)

// ExtractProfile loads a CV file and returns compact text suitable for matching
// prompts/scoring. Supported formats: plain text and PDF.
func ExtractProfile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("cv path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("checking cv file metadata: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("cv path points to a directory")
	}

	signature, err := readSignature(path)
	if err != nil {
		return "", err
	}

	switch {
	case bytes.HasPrefix(signature, pdfHeader):
		return extractPDF(path)
	case bytes.HasPrefix(signature, zipHeader):
		return "", fmt.Errorf("%w: zip-based documents (for example .docx) are not supported; use PDF or plain text", ErrUnsupportedFormat)
	case bytes.HasPrefix(signature, docHeader):
		return "", fmt.Errorf("%w: legacy .doc documents are not supported; use PDF or plain text", ErrUnsupportedFormat)
	default:
		return extractText(path)
	}
}

func readSignature(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening cv file for signature check: %w", err)
	}
	defer file.Close()

	buf := make([]byte, maxSigRead)
	n, err := io.ReadFull(file, buf)
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return buf[:n], nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading cv file signature: %w", err)
	}
	return buf[:n], nil
}

func extractText(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading cv file: %w", err)
	}
	if len(raw) == 0 {
		return "", errors.New("cv file is empty")
	}
	if len(raw) > maxTextFileBytes {
		raw = raw[:maxTextFileBytes]
	}
	if bytes.HasPrefix(raw, bomUTF8) {
		raw = raw[len(bomUTF8):]
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return "", fmt.Errorf("%w: binary file content detected", ErrUnsupportedFormat)
	}

	text := normalize(string(raw))
	if text == "" {
		return "", ErrNoUsableText
	}
	return truncateRunes(text, maxProfileRunes), nil
}

func extractPDF(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening pdf cv file: %w", err)
	}
	defer file.Close()

	plainTextReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("extracting pdf text: %w", err)
	}

	raw, err := io.ReadAll(io.LimitReader(plainTextReader, maxExtractedPDFTextByte))
	if err != nil {
		return "", fmt.Errorf("reading extracted pdf text: %w", err)
	}

	text := normalize(string(raw))
	if text == "" {
		return "", ErrNoUsableText
	}
	return truncateRunes(text, maxProfileRunes), nil
}

func normalize(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.Join(strings.Fields(value), " ")
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
