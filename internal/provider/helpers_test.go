package provider

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func testCheckLabelMap(name string, partialKey string, expectedLabels map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		attrs := s.RootModule().Resources[name].Primary.Attributes
		labelMap := getLabelMapForPartialKey(attrs, partialKey)

		if len(labelMap) != len(expectedLabels) {
			return fmt.Errorf("expected %v labels, found %v", len(expectedLabels), len(labelMap))
		}

		for l, v := range expectedLabels {
			if labelMap[l] != v {
				return fmt.Errorf("expected value %v for label %v, got %v", v, l, labelMap[v])
			}
		}

		return nil
	}
}

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
		{10000000000, "10.0"},
		{100000000000, "100.0"},
		{10000000, "0.01"},
		{50000000, "0.05"},
		{500000000, "0.5"},
		{1, "0.000000001"},
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

func TestExpandGPUDeviceRequests(t *testing.T) {
	const (
		gpuUUID1 = "GPU-3a23c669-1f69-c64e-cf85-44e9b07e7a2a"
		gpuUUID2 = "GPU-8b7d2e41-5c0a-4f3e-9d61-2a4b6c8e0f13"
	)

	deviceIDsRequest := func(ids ...string) []container.DeviceRequest {
		return []container.DeviceRequest{{
			DeviceIDs:    ids,
			Capabilities: [][]string{{"gpu"}},
			Options:      map[string]string{},
		}}
	}

	tests := []struct {
		name     string
		input    string
		expected []container.DeviceRequest
	}{
		{
			name:  "all",
			input: "all",
			expected: []container.DeviceRequest{{
				Count:        -1,
				Capabilities: [][]string{{"gpu"}},
				Options:      map[string]string{},
			}},
		},
		{
			name:     "single device index",
			input:    "device=0",
			expected: deviceIDsRequest("0"),
		},
		{
			name:     "single device uuid",
			input:    "device=" + gpuUUID1,
			expected: deviceIDsRequest(gpuUUID1),
		},
		{
			name:     "multiple device indexes",
			input:    "device=0,2",
			expected: deviceIDsRequest("0", "2"),
		},
		{
			name:     "multiple device uuids",
			input:    "device=" + gpuUUID1 + "," + gpuUUID2,
			expected: deviceIDsRequest(gpuUUID1, gpuUUID2),
		},
		{
			name:     "double quoted multiple device indexes",
			input:    "\"device=0,2\"",
			expected: deviceIDsRequest("0", "2"),
		},
		{
			name:     "double wrapped multiple device indexes",
			input:    "'\"device=0,2\"'",
			expected: deviceIDsRequest("0", "2"),
		},
		{
			name:  "device with other options",
			input: "device=0,driver=nvidia",
			expected: []container.DeviceRequest{{
				Driver:       "nvidia",
				DeviceIDs:    []string{"0"},
				Capabilities: [][]string{{"gpu"}},
				Options:      map[string]string{},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := expandGPUDeviceRequests(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !reflect.DeepEqual(result, test.expected) {
				t.Fatalf("expected %+v, got %+v", test.expected, result)
			}
		})
	}
}

func TestExpandGPUDeviceRequestsRoundTrip(t *testing.T) {
	for _, value := range []string{
		"all",
		"device=0",
		"device=GPU-3a23c669-1f69-c64e-cf85-44e9b07e7a2a",
		"device=0,2",
		"device=GPU-3a23c669-1f69-c64e-cf85-44e9b07e7a2a,GPU-8b7d2e41-5c0a-4f3e-9d61-2a4b6c8e0f13",
	} {
		t.Run(value, func(t *testing.T) {
			deviceRequests, err := expandGPUDeviceRequests(value)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			result, ok := flattenGPUsFromDeviceRequests(deviceRequests)
			if result != value || !ok {
				t.Fatalf("expected (%q, true), got (%q, %t)", value, result, ok)
			}
		})
	}
}

func TestContainerCpusSuppressesEquivalentValues(t *testing.T) {
	suppress := resourceDockerContainer().Schema["cpus"].DiffSuppressFunc
	if suppress == nil {
		t.Fatal("expected cpus to have a DiffSuppressFunc")
	}

	tests := []struct {
		old, new string
		expected bool
	}{
		{"2.0", "2", true},
		{"0.5", ".5", true},
		{"1.5", "1.50", true},
		{"1.5", "2", false},
		{"", "2", false},
	}

	for _, test := range tests {
		if got := suppress("cpus", test.old, test.new, nil); got != test.expected {
			t.Errorf("suppress(%q, %q) = %v, want %v", test.old, test.new, got, test.expected)
		}
	}
}
