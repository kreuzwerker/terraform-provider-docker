package provider

import (
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/container"
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
