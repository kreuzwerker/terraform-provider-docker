package provider

import "testing"

func TestParseOptionalPlatform(t *testing.T) {
	t.Run("empty value", func(t *testing.T) {
		platformValue, err := parseOptionalPlatform("")
		if err != nil {
			t.Fatalf("parseOptionalPlatform returned error: %s", err)
		}
		if platformValue != nil {
			t.Fatalf("expected nil platform for empty value")
		}
	})

	t.Run("valid value", func(t *testing.T) {
		platformValue, err := parseOptionalPlatform("linux/amd64")
		if err != nil {
			t.Fatalf("parseOptionalPlatform returned error: %s", err)
		}
		if platformValue == nil {
			t.Fatalf("expected parsed platform, got nil")
		}
		if platformValue.OS != "linux" || platformValue.Architecture != "amd64" {
			t.Fatalf("unexpected parsed platform: %#v", platformValue)
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		_, err := parseOptionalPlatform("not a platform")
		if err == nil {
			t.Fatalf("expected parseOptionalPlatform to return an error")
		}
	})
}
