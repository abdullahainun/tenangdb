package compression

import (
	"archive/tar"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

func newTestCompressor(t *testing.T, format string) *Compressor {
	t.Helper()
	cfg := &config.CompressionConfig{
		Enabled:      true,
		Format:       format,
		Level:        6,
		KeepOriginal: false,
	}
	log := logger.NewLogger("error")
	return NewCompressor(cfg, log)
}

func createTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), bytes.Repeat([]byte{0x42}, 1000), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func verifyContent(t *testing.T, decompressedDir string) {
	t.Helper()
	checkFileContent(t, filepath.Join(decompressedDir, "test.txt"), "hello world")
	checkFileContent(t, filepath.Join(decompressedDir, "sub", "nested.txt"), "nested")
	data, err := os.ReadFile(filepath.Join(decompressedDir, "data.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1000 {
		t.Fatalf("expected 1000 bytes, got %d", len(data))
	}
}

func checkFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatalf("%s: expected %q, got %q", path, expected, string(data))
	}
}

func identifyCompression(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	magic := make([]byte, 6)
	if _, err := f.Read(magic); err != nil {
		return "", err
	}

	switch {
	case magic[0] == 0x1f && magic[1] == 0x8b:
		return "gzip", nil
	case magic[0] == 0x28 && magic[1] == 0xb5 && magic[2] == 0x2f && magic[3] == 0xfd:
		return "zstd", nil
	case magic[0] == 0xfd && magic[1] == 0x37 && magic[2] == 0x7a && magic[3] == 0x58 && magic[4] == 0x5a:
		return "xz", nil
	default:
		return "unknown", nil
	}
}

func TestCompressAndDecompressTarGz(t *testing.T) {
	c := newTestCompressor(t, "tar.gz")
	src := createTestDir(t)
	out, err := c.CompressBackup(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, ".tar.gz") {
		t.Fatalf("expected .tar.gz suffix, got %s", out)
	}

	typ, err := identifyCompression(out)
	if err != nil {
		t.Fatal(err)
	}
	if typ != "gzip" {
		t.Fatalf("expected gzip compression, got %s", typ)
	}

	// verify round-trip
	dir, err := c.DecompressBackup(out)
	if err != nil {
		t.Fatal(err)
	}
	verifyContent(t, dir)
}

func TestCompressAndDecompressTarZst(t *testing.T) {
	c := newTestCompressor(t, "tar.zst")
	src := createTestDir(t)
	out, err := c.CompressBackup(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, ".tar.zst") {
		t.Fatalf("expected .tar.zst suffix, got %s", out)
	}

	typ, err := identifyCompression(out)
	if err != nil {
		t.Fatal(err)
	}
	if typ != "zstd" {
		t.Fatalf("expected zstd compression, got %s", typ)
	}

	dir, err := c.DecompressBackup(out)
	if err != nil {
		t.Fatal(err)
	}
	verifyContent(t, dir)
}

func TestCompressAndDecompressTarXz(t *testing.T) {
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz binary not found, skipping")
	}

	c := newTestCompressor(t, "tar.xz")
	src := createTestDir(t)
	out, err := c.CompressBackup(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out, ".tar.xz") {
		t.Fatalf("expected .tar.xz suffix, got %s", out)
	}

	typ, err := identifyCompression(out)
	if err != nil {
		t.Fatal(err)
	}
	if typ != "xz" {
		t.Fatalf("expected xz compression, got %s", typ)
	}

	dir, err := c.DecompressBackup(out)
	if err != nil {
		t.Fatal(err)
	}
	verifyContent(t, dir)
}

func TestNotGzip(t *testing.T) {
	// verify zst and xz are NOT gzip
	tests := []struct {
		format string
		skip   string
	}{
		{"tar.zst", ""},
		{"tar.xz", "xz binary not found, skipping"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			if strings.HasPrefix(tt.skip, "xz") {
				if _, err := exec.LookPath("xz"); err != nil {
					t.Skip(tt.skip)
				}
			}
			c := newTestCompressor(t, tt.format)
			src := createTestDir(t)
			out, err := c.CompressBackup(src)
			if err != nil {
				t.Fatal(err)
			}

			// ensure it's not gzip
			f, err := os.Open(out)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			magic := make([]byte, 2)
			if _, err := f.Read(magic); err != nil {
				t.Fatal(err)
			}
			if magic[0] == 0x1f && magic[1] == 0x8b {
				t.Fatal("output is gzip — format not properly implemented")
			}
		})
	}
}

func TestTarGzRoundTripLarge(t *testing.T) {
	c := newTestCompressor(t, "tar.gz")
	src := t.TempDir()

	// create a larger file to test streaming
	data := make([]byte, 100000)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if err := os.WriteFile(filepath.Join(src, "large.bin"), data, 0644); err != nil {
		t.Fatal(err)
	}

	out, err := c.CompressBackup(src)
	if err != nil {
		t.Fatal(err)
	}

	dir, err := c.DecompressBackup(out)
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("large file content mismatch after round-trip")
	}
}

func TestUnsupportedFormat(t *testing.T) {
	c := newTestCompressor(t, "tar.bz2")
	src := createTestDir(t)
	_, err := c.CompressBackup(src)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

// Test the writeTarTo helper produces valid tar
func TestWriteTarTo(t *testing.T) {
	c := newTestCompressor(t, "tar.gz")
	src := createTestDir(t)

	var buf bytes.Buffer
	if err := c.writeTarTo(src, &buf); err != nil {
		t.Fatal(err)
	}

	// verify it's valid tar
	tr := tar.NewReader(&buf)
	seen := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		seen[hdr.Name] = true
	}
	if !seen["test.txt"] {
		t.Fatal("tar missing test.txt")
	}
	if !seen["sub/nested.txt"] {
		t.Fatal("tar missing sub/nested.txt")
	}
}
