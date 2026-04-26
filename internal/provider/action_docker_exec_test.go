package provider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvironmentVariables(t *testing.T) {
	t.Setenv("ENV_TEST_KEY", "test-value")

	got := resolveEnvironmentVariables([]string{"ENV_TEST_KEY", "EXPLICIT=value", "MISSING_KEY"})

	expected := []string{"ENV_TEST_KEY=test-value", "EXPLICIT=value", "MISSING_KEY="}
	if len(got) != len(expected) {
		t.Fatalf("expected %d env vars, got %d", len(expected), len(got))
	}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected env at index %d to be %q, got %q", i, expected[i], got[i])
		}
	}
}

func TestParseEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	envFilePath := filepath.Join(tempDir, "test.env")

	content := "\n# comment\nFOO=bar\n\nBAZ=qux\n"
	if err := os.WriteFile(envFilePath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write env file: %s", err)
	}

	got, err := parseEnvFile(envFilePath)
	if err != nil {
		t.Fatalf("parseEnvFile returned error: %s", err)
	}

	expected := []string{"FOO=bar", "BAZ=qux"}
	if len(got) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(got))
	}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected env line at index %d to be %q, got %q", i, expected[i], got[i])
		}
	}
}
