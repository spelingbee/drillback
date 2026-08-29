package loader

import (
	"os"
	"path/filepath"
	"testing"
)

// The format is detected from the file's first bytes, never from its extension,
// because the extension is what the backup script's author believed and the bytes are
// what the backup actually contains.
func TestDetectFormat(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
		want    string
	}{
		{"a custom-format dump", []byte("PGDMP\x01\x0e\x00"), FormatCustom},
		{"a plain SQL dump", []byte("--\n-- PostgreSQL database dump\n--\n\nSET statement_timeout = 0;\n"), FormatPlain},
		{"an HTML error page where a dump should be", []byte("<!DOCTYPE html>\n<html><body>502 Bad Gateway"), FormatPlain},
		{"an empty file", []byte{}, FormatEmpty},
		{"a file shorter than the magic string", []byte("PG"), FormatPlain},
		{"a gzip stream that was never decompressed", []byte{0x1f, 0x8b, 0x08, 0x00, 0x00}, FormatPlain},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "dump")
			if err := os.WriteFile(path, tc.content, 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := DetectFormat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("DetectFormat = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetectFormatOnAMissingFile(t *testing.T) {
	if _, err := DetectFormat(filepath.Join(t.TempDir(), "nothing")); err == nil {
		t.Fatal("a missing dump must be an error, not a format")
	}
}
