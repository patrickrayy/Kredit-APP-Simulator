package handler

import (
	"strings"
	"testing"
)

func TestIsValidDocType(t *testing.T) {
	tests := []struct {
		docType string
		want    bool
	}{
		{"ktp", true},
		{"npwp", true},
		{"slip_gaji", true},
		{"rekening_koran", true},
		{"lainnya", true},
		{"paspor", false},
		{"", false},
		{"KTP", false}, // handler yang menormalisasi ke huruf kecil, bukan fungsi ini
	}

	for _, tt := range tests {
		t.Run(tt.docType, func(t *testing.T) {
			if got := isValidDocType(tt.docType); got != tt.want {
				t.Errorf("isValidDocType(%q) = %v, want %v", tt.docType, got, tt.want)
			}
		})
	}
}

func TestSafeExt(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"simple pdf", "ktp.pdf", ".pdf"},
		{"uppercase lowered", "FOTO.JPG", ".jpg"},
		{"multiple dots uses last", "arsip.tar.gz", ".gz"},
		{"no extension", "dokumen", ""},
		{"trailing dot", "dokumen.", ""},
		{"traversal unix", "../../etc/passwd", ""},
		{"traversal windows keeps only ext", `..\..\cmd\api\main.go`, ".go"},
		{"space in ext rejected", "file.p df", ""},
		{"too long ext rejected", "file.abcdefghijklmnop", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeExt(tt.filename); got != tt.want {
				t.Errorf("safeExt(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestStoredFileName_IsRandomAndHasNoPath(t *testing.T) {
	a := storedFileName(`..\..\evil.pdf`)
	b := storedFileName(`..\..\evil.pdf`)

	if a == b {
		t.Fatalf("expected different names for two uploads, both got %q", a)
	}
	if strings.ContainsAny(a, `/\`) || strings.Contains(a, "..") {
		t.Fatalf("stored name %q must not contain path elements", a)
	}
	if !strings.HasSuffix(a, ".pdf") {
		t.Fatalf("stored name %q should keep the .pdf extension", a)
	}
}
