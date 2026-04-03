package provider

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

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

func TestFlattenGPUsFromDeviceRequests(t *testing.T) {
	tests := []struct {
		name           string
		deviceRequests []container.DeviceRequest
		expectedValue  string
		expectedOK     bool
	}{
		{
			name:           "empty requests",
			deviceRequests: []container.DeviceRequest{},
			expectedValue:  "",
			expectedOK:     true,
		},
		{
			name: "single all gpu request",
			deviceRequests: []container.DeviceRequest{{
				Count:        -1,
				Capabilities: [][]string{{"gpu"}},
			}},
			expectedValue: "all",
			expectedOK:    true,
		},
		{
			name: "single all gpu request with uppercase capability",
			deviceRequests: []container.DeviceRequest{{
				Count:        -1,
				Capabilities: [][]string{{"GPU"}},
			}},
			expectedValue: "all",
			expectedOK:    true,
		},
		{
			name: "single request with specific device ids",
			deviceRequests: []container.DeviceRequest{{
				Count:        0,
				DeviceIDs:    []string{"0", "2"},
				Capabilities: [][]string{{"gpu"}},
			}},
			expectedValue: "device=0,2",
			expectedOK:    true,
		},
		{
			name: "single request with gpu uuid",
			deviceRequests: []container.DeviceRequest{{
				Count:        0,
				DeviceIDs:    []string{"GPU-3a23c669-1f69-c64e-cf85-44e9b07e7a2a"},
				Capabilities: [][]string{{"gpu"}},
			}},
			expectedValue: "device=GPU-3a23c669-1f69-c64e-cf85-44e9b07e7a2a",
			expectedOK:    true,
		},
		{
			name: "single request without gpu capability",
			deviceRequests: []container.DeviceRequest{{
				Count:        -1,
				Capabilities: [][]string{{"compute"}},
			}},
			expectedValue: "",
			expectedOK:    false,
		},
		{
			name: "multiple requests",
			deviceRequests: []container.DeviceRequest{
				{
					Count:        -1,
					Capabilities: [][]string{{"gpu"}},
				},
				{
					Count:        -1,
					Capabilities: [][]string{{"gpu"}},
				},
			},
			expectedValue: "",
			expectedOK:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultValue, resultOK := flattenGPUsFromDeviceRequests(test.deviceRequests)
			if resultValue != test.expectedValue || resultOK != test.expectedOK {
				t.Fatalf("expected (%q, %t), got (%q, %t)", test.expectedValue, test.expectedOK, resultValue, resultOK)
			}
		})
	}
}

func TestNormalizeGPUOptionString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "raw value",
			input:    "device=0,2",
			expected: "device=0,2",
		},
		{
			name:     "double quoted value",
			input:    "\"device=0,2\"",
			expected: "device=0,2",
		},
		{
			name:     "single quoted value",
			input:    "'device=0,2'",
			expected: "device=0,2",
		},
		{
			name:     "double wrapped value",
			input:    "'\"device=0,2\"'",
			expected: "device=0,2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := normalizeGPUOptionString(test.input)
			if result != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, result)
			}
		})
	}
}
