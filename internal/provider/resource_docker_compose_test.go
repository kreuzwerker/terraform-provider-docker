package provider

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func TestComposeVersionLabel_FallbackToBuildInfo(t *testing.T) {
	got := composeVersionLabelFor("", debug.ReadBuildInfo)
	if got == "" {
		t.Fatal("compose version label must not be empty")
	}

	expected := "unknown"
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range buildInfo.Deps {
			if dep.Path == "github.com/docker/compose/v2" {
				expected = strings.TrimPrefix(dep.Version, "v")
				break
			}
		}
	}

	if got != expected {
		t.Fatalf("compose version label = %q, want %q", got, expected)
	}
}

func TestHashComposeContentFiles_StableDigest(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "compose.yaml")
	envPath := filepath.Join(dir, "app.env")

	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx\n"), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	paths := []string{composePath, envPath}
	first, err := hashComposeContentFiles(paths)
	if err != nil {
		t.Fatalf("hashComposeContentFiles() error = %v", err)
	}
	second, err := hashComposeContentFiles(paths)
	if err != nil {
		t.Fatalf("hashComposeContentFiles() second call error = %v", err)
	}
	if first != second {
		t.Fatalf("digest not stable: %q != %q", first, second)
	}

	sum := sha256.New()
	var lengthBuf [8]byte
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %q: %v", path, err)
		}
		binary.BigEndian.PutUint64(lengthBuf[:], uint64(len(data)))
		sum.Write(lengthBuf[:])
		sum.Write(data)
	}
	want := hex.EncodeToString(sum.Sum(nil))
	if first != want {
		t.Fatalf("digest = %q, want %q", first, want)
	}
}

func TestHashComposeContentFiles_OrderMatters(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(a, []byte("aaa"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(b, []byte("bbb"), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	ab, err := hashComposeContentFiles([]string{a, b})
	if err != nil {
		t.Fatalf("hash a,b: %v", err)
	}
	ba, err := hashComposeContentFiles([]string{b, a})
	if err != nil {
		t.Fatalf("hash b,a: %v", err)
	}
	if ab == ba {
		t.Fatalf("expected order-sensitive digests, both were %q", ab)
	}
}

func TestHashComposeContentFiles_MissingFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "compose.yaml")
	if err := os.WriteFile(existing, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write existing: %v", err)
	}

	_, err := hashComposeContentFiles([]string{existing, filepath.Join(dir, "missing.env")})
	if err == nil {
		t.Fatal("expected error for missing listed file")
	}
	if !strings.Contains(err.Error(), "missing.env") {
		t.Fatalf("error = %v, want path mention", err)
	}
}
