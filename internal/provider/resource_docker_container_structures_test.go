package provider

import (
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestVolumeSetToDockerVolumes_withSELinuxRelabelModes(t *testing.T) {
	volumeResource := resourceDockerContainer().Schema["volumes"].Elem.(*schema.Resource)
	hash := schema.HashResource(volumeResource)

	testCases := []struct {
		name     string
		readOnly bool
		relabel  string
		expected string
	}{
		{
			name:     "without_relabel",
			readOnly: false,
			relabel:  "",
			expected: "/host/path:/container/path:rw",
		},
		{
			name:     "shared_relabel",
			readOnly: true,
			relabel:  "z",
			expected: "/host/path:/container/path:ro,z",
		},
		{
			name:     "private_relabel",
			readOnly: false,
			relabel:  "Z",
			expected: "/host/path:/container/path:rw,Z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			volumes := schema.NewSet(hash, []interface{}{
				map[string]interface{}{
					"from_container":  "",
					"container_path":  "/container/path",
					"host_path":       "/host/path",
					"volume_name":     "",
					"read_only":       tc.readOnly,
					"selinux_relabel": tc.relabel,
				},
			})

			_, binds, volumesFrom, err := volumeSetToDockerVolumes(volumes)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if len(volumesFrom) != 0 {
				t.Fatalf("expected no volumes_from entries, got %d", len(volumesFrom))
			}
			if len(binds) != 1 {
				t.Fatalf("expected one bind entry, got %d", len(binds))
			}
			if binds[0] != tc.expected {
				t.Fatalf("expected bind %q, got %q", tc.expected, binds[0])
			}
		})
	}
}

func TestContainerLogOptsForState(t *testing.T) {
	containerLogOpts := map[string]string{
		"max-size": "100m",
		"max-file": "3",
	}

	testCases := []struct {
		name        string
		rawConfig   map[string]interface{}
		wantLogOpts map[string]string
	}{
		{
			name:        "log_opts omitted from configuration",
			rawConfig:   map[string]interface{}{},
			wantLogOpts: nil,
		},
		{
			name: "log_opts configured",
			rawConfig: map[string]interface{}{
				"log_opts": map[string]interface{}{
					"max-size": "100m",
					"max-file": "3",
				},
			},
			wantLogOpts: containerLogOpts,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceData := schema.TestResourceDataRaw(t, resourceDockerContainer().Schema, tc.rawConfig)
			got := containerLogOptsForState(resourceData, containerLogOpts)

			if !reflect.DeepEqual(tc.wantLogOpts, got) {
				t.Fatalf("expected log opts %#v, got %#v", tc.wantLogOpts, got)
			}
		})
	}
}

func TestFlattenDevices(t *testing.T) {
	singleDeviceMappings := []container.DeviceMapping{
		{
			PathOnHost:        "/dev/test0",
			PathInContainer:   "/dev/container0",
			CgroupPermissions: "rwm",
		},
	}

	t.Run("does not panic when configured devices are empty", func(t *testing.T) {
		got := flattenDevices(singleDeviceMappings, schema.NewSet(schema.HashString, []interface{}{}))
		if len(got) != 1 {
			t.Fatalf("expected 1 device, got %d", len(got))
		}

		deviceMap := got[0].(map[string]interface{})
		if _, ok := deviceMap["container_path"]; ok {
			t.Fatalf("expected container_path to be omitted when not explicitly configured")
		}
	})

	t.Run("keeps container_path when explicitly configured", func(t *testing.T) {
		deviceResource := resourceDockerContainer().Schema["devices"].Elem.(*schema.Resource)
		hash := schema.HashResource(deviceResource)
		configuredDevices := schema.NewSet(hash, []interface{}{
			map[string]interface{}{
				"host_path":      "/dev/test0",
				"container_path": "/dev/container0",
				"permissions":    "rwm",
			},
		})

		got := flattenDevices(singleDeviceMappings, configuredDevices)
		if len(got) != 1 {
			t.Fatalf("expected 1 device, got %d", len(got))
		}

		deviceMap := got[0].(map[string]interface{})
		if deviceMap["container_path"] != "/dev/container0" {
			t.Fatalf("expected container_path to be preserved, got %#v", deviceMap["container_path"])
		}
	})

	t.Run("matches configured devices by host_path instead of index", func(t *testing.T) {
		deviceMappings := []container.DeviceMapping{
			{
				PathOnHost:        "/dev/test0",
				PathInContainer:   "/dev/container0",
				CgroupPermissions: "rwm",
			},
			{
				PathOnHost:        "/dev/test1",
				PathInContainer:   "/dev/container1",
				CgroupPermissions: "rwm",
			},
		}

		deviceResource := resourceDockerContainer().Schema["devices"].Elem.(*schema.Resource)
		hash := schema.HashResource(deviceResource)
		configuredDevices := schema.NewSet(hash, []interface{}{
			map[string]interface{}{
				"host_path":      "/dev/test1",
				"container_path": "/dev/container1",
				"permissions":    "rwm",
			},
		})

		got := flattenDevices(deviceMappings, configuredDevices)

		first := got[0].(map[string]interface{})
		if _, ok := first["container_path"]; ok {
			t.Fatalf("expected first device to omit container_path when not configured")
		}

		second := got[1].(map[string]interface{})
		if second["container_path"] != "/dev/container1" {
			t.Fatalf("expected second device container_path to be preserved, got %#v", second["container_path"])
		}
	})
}

func TestNormalizePortIP(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty defaults to 0.0.0.0", input: "", expected: "0.0.0.0"},
		{name: "ipv4 unchanged", input: "0.0.0.0", expected: "0.0.0.0"},
		{name: "ipv4 specific unchanged", input: "192.168.1.1", expected: "192.168.1.1"},
		{name: "ipv6 bare unchanged", input: "::", expected: "::"},
		{name: "ipv6 bracketed stripped", input: "[::]", expected: "::"},
		{name: "ipv6 full bracketed stripped", input: "[::1]", expected: "::1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizePortIP(tc.input)
			if got != tc.expected {
				t.Fatalf("normalizePortIP(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestFlattenContainerPorts(t *testing.T) {
	t.Run("normalizes IPv6 IPs from API", func(t *testing.T) {
		portMap := nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "80"},
				{HostIP: "::", HostPort: "80"},
			},
		}

		got := flattenContainerPorts(portMap)
		if len(got) != 2 {
			t.Fatalf("expected 2 ports, got %d", len(got))
		}

		first := got[0].(map[string]interface{})
		if first["ip"] != "0.0.0.0" {
			t.Fatalf("expected first port ip to be 0.0.0.0, got %q", first["ip"])
		}
		second := got[1].(map[string]interface{})
		if second["ip"] != "::" {
			t.Fatalf("expected second port ip to be ::, got %q", second["ip"])
		}
	})

	t.Run("sorts bindings by IP within same port", func(t *testing.T) {
		portMap := nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{HostIP: "::", HostPort: "80"},
				{HostIP: "0.0.0.0", HostPort: "80"},
			},
		}

		got := flattenContainerPorts(portMap)
		if len(got) != 2 {
			t.Fatalf("expected 2 ports, got %d", len(got))
		}

		first := got[0].(map[string]interface{})
		if first["ip"] != "0.0.0.0" {
			t.Fatalf("expected first port ip to be 0.0.0.0 (sorted), got %q", first["ip"])
		}
		second := got[1].(map[string]interface{})
		if second["ip"] != "::" {
			t.Fatalf("expected second port ip to be :: (sorted), got %q", second["ip"])
		}
	})

	t.Run("sorts by port then by IP", func(t *testing.T) {
		portMap := nat.PortMap{
			"443/tcp": []nat.PortBinding{
				{HostIP: "::", HostPort: "443"},
				{HostIP: "0.0.0.0", HostPort: "443"},
			},
			"80/tcp": []nat.PortBinding{
				{HostIP: "::", HostPort: "80"},
				{HostIP: "0.0.0.0", HostPort: "80"},
			},
		}

		got := flattenContainerPorts(portMap)
		if len(got) != 4 {
			t.Fatalf("expected 4 ports, got %d", len(got))
		}

		expected := []struct {
			internal int
			ip       string
		}{
			{80, "0.0.0.0"},
			{80, "::"},
			{443, "0.0.0.0"},
			{443, "::"},
		}

		for i, exp := range expected {
			m := got[i].(map[string]interface{})
			if m["internal"] != exp.internal || m["ip"] != exp.ip {
				t.Fatalf("port[%d]: expected internal=%d ip=%q, got internal=%v ip=%v",
					i, exp.internal, exp.ip, m["internal"], m["ip"])
			}
		}
	})
}
