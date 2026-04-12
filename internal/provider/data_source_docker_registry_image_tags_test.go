package provider

import "testing"

func TestFilterStrictSemverTags(t *testing.T) {
	tags := []string{
		"latest",
		"1.2.3",
		"1.2.4-rc.1",
		"1.2.5+build.7",
		"2024-01-01",
		"v2.0.0",
	}

	filtered := filterStrictSemverTags(tags)

	if len(filtered) != 3 {
		t.Fatalf("expected 3 semver tags, got %d: %#v", len(filtered), filtered)
	}

	if filtered[0] != "1.2.3" || filtered[1] != "1.2.5+build.7" || filtered[2] != "v2.0.0" {
		t.Fatalf("unexpected filtered tags: %#v", filtered)
	}
}
