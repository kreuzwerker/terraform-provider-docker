package provider

import "testing"

func TestNanoInt64ToDecimalString(t *testing.T) {
	tests := []struct {
		nanoInt64 int64
		expected  string
	}{
		{0, "0"},
		{1000000000, "1.0"},
		{1500000000, "1.5"},
		{2000000000, "2.0"},
		{2250000000, "2.25"},
	}

	for _, test := range tests {
		result := nanoInt64ToDecimalString(test.nanoInt64)
		if result != test.expected {
			t.Errorf("Expected %s for %d, got %s", test.expected, test.nanoInt64, result)
		}
	}
}
