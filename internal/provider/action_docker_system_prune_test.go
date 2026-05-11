package provider

import "testing"

func TestParseSystemPruneFilterExpressions(t *testing.T) {
	t.Run("valid filters", func(t *testing.T) {
		result, err := parseSystemPruneFilterExpressions([]string{
			"label=app=demo",
			"until=24h",
		})
		if err != nil {
			t.Fatalf("parseSystemPruneFilterExpressions returned error: %s", err)
		}

		labels := result.Get("label")
		if len(labels) != 1 || labels[0] != "app=demo" {
			t.Fatalf("expected label filter app=demo, got %#v", labels)
		}

		untilValues := result.Get("until")
		if len(untilValues) != 1 || untilValues[0] != "24h" {
			t.Fatalf("expected until filter 24h, got %#v", untilValues)
		}
	})

	t.Run("invalid filter without separator", func(t *testing.T) {
		_, err := parseSystemPruneFilterExpressions([]string{"invalid-filter"})
		if err == nil {
			t.Fatalf("expected error for invalid filter expression")
		}
	})

	t.Run("invalid empty key", func(t *testing.T) {
		_, err := parseSystemPruneFilterExpressions([]string{"=value"})
		if err == nil {
			t.Fatalf("expected error for filter with empty key")
		}
	})
}
