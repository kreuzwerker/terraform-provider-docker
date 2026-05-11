package provider

import (
	"testing"
)

func TestNormalizeRegistryAddress(t *testing.T) {
	t.Run("Should return same address if http:// is used", func(t *testing.T) {
		address := "http://registry.com"
		expected := "http://registry.com"
		actual := normalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})
	t.Run("Should return https address if no protocol is specified", func(t *testing.T) {
		address := "registry.com"
		expected := "https://registry.com"
		actual := normalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})
	t.Run("Should return https address if https protocol is specified", func(t *testing.T) {
		address := "https://registry.com"
		expected := "https://registry.com"
		actual := normalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})
}

func TestCanonicalizeRegistryAddress(t *testing.T) {
	t.Run("Should keep host-only https address unchanged", func(t *testing.T) {
		address := "https://registry.com"
		expected := "https://registry.com"
		actual := canonicalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})

	t.Run("Should strip repository path from https address", func(t *testing.T) {
		address := "https://registry.com/team/project"
		expected := "https://registry.com"
		actual := canonicalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})

	t.Run("Should strip repository path from http address", func(t *testing.T) {
		address := "http://registry.com/team/project"
		expected := "http://registry.com"
		actual := canonicalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})

	t.Run("Should normalize host without protocol", func(t *testing.T) {
		address := "registry.com/team/project"
		expected := "https://registry.com"
		actual := canonicalizeRegistryAddress(address)
		if actual != expected {
			t.Fatalf("Expected %s, got %s", expected, actual)
		}
	})
}
