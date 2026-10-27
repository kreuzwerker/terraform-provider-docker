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

func TestVolumeSetToDockerVolumes_HostPathPrecedence(t *testing.T) {
	volumeResource := resourceDockerContainer().Schema["volumes"].Elem.(*schema.Resource)
	hash := schema.HashResource(volumeResource)

	volumes := schema.NewSet(hash, []interface{}{
		map[string]interface{}{
			"from_container":  "",
			"container_path":  "/container/path",
			"host_path":       "/host/path",
			"volume_name":     "named-volume",
			"read_only":       false,
			"selinux_relabel": "",
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
	if binds[0] != "/host/path:/container/path:rw" {
		t.Fatalf("expected host_path bind to be used, got %q", binds[0])
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

func TestDeviceCgroupRulesSchemaAndConversion(t *testing.T) {
	schemaField, ok := resourceDockerContainer().Schema["device_cgroup_rules"]
	if !ok {
		t.Fatalf("expected device_cgroup_rules to be present in schema")
	}
	if schemaField.Type != schema.TypeSet {
		t.Fatalf("expected device_cgroup_rules to be a set, got %v", schemaField.Type)
	}
	if elem, ok := schemaField.Elem.(*schema.Schema); !ok || elem.Type != schema.TypeString {
		t.Fatalf("expected device_cgroup_rules elements to be strings")
	}

	resourceData := schema.TestResourceDataRaw(t, resourceDockerContainer().Schema, map[string]interface{}{
		"device_cgroup_rules": []interface{}{"c 189:* rmw"},
	})
	value := resourceData.Get("device_cgroup_rules").(*schema.Set)

	got := stringSetToStringSlice(value)
	if !reflect.DeepEqual(got, []string{"c 189:* rmw"}) {
		t.Fatalf("unexpected device_cgroup_rules conversion: %#v", got)
	}

	if flattened := flattenDeviceCgroupRules([]string{"c 189:* rmw"}); !reflect.DeepEqual(flattened, []interface{}{"c 189:* rmw"}) {
		t.Fatalf("unexpected device_cgroup_rules flattening: %#v", flattened)
	}
}

func TestSupportsCreateTimeNetworkEndpoints(t *testing.T) {
	testCases := []struct {
		apiVersion  string
		networkMode container.NetworkMode
		expected    bool
	}{
		{apiVersion: "1.43", networkMode: "bridge", expected: false},
		{apiVersion: "1.44", networkMode: "bridge", expected: true},
		{apiVersion: "1.51", networkMode: "default", expected: true},
		{apiVersion: "1.51", networkMode: "my-network", expected: true},
		{apiVersion: "1.51", networkMode: "host", expected: false},
		{apiVersion: "1.51", networkMode: "none", expected: false},
		{apiVersion: "1.51", networkMode: "container:foo", expected: false},
	}

	for _, tc := range testCases {
		if got := supportsCreateTimeNetworkEndpoints(tc.apiVersion, tc.networkMode); got != tc.expected {
			t.Errorf("supportsCreateTimeNetworkEndpoints(%q, %q) = %t, expected %t", tc.apiVersion, tc.networkMode, got, tc.expected)
		}
	}
}

func TestExpandContainerCreateEndpointsConfig(t *testing.T) {
	networksAdvanced := resourceDockerContainer().Schema["networks_advanced"].Elem.(*schema.Resource)
	rawNetwork := func(name, macAddress string) map[string]interface{} {
		return map[string]interface{}{
			"name":           name,
			"aliases":        schema.NewSet(schema.HashString, []interface{}{"alias-" + name}),
			"ipv4_address":   "10.0.0.2",
			"ipv6_address":   "",
			"link_local_ips": schema.NewSet(schema.HashString, []interface{}{}),
			"mac_address":    macAddress,
			"driver_opts":    schema.NewSet(schema.HashString, []interface{}{"opt=value"}),
			"gw_priority":    10,
		}
	}
	networks := schema.NewSet(schema.HashResource(networksAdvanced), []interface{}{
		rawNetwork("net-1", "02:42:ac:11:00:aa"),
		rawNetwork("net-2", ""),
	}).List()

	t.Run("bridge_mode", func(t *testing.T) {
		endpoints, err := expandContainerCreateEndpointsConfig("bridge", networks)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if len(endpoints) != 2 {
			t.Fatalf("expected 2 endpoints, got %d: %#v", len(endpoints), endpoints)
		}
		endpoint := endpoints["net-1"]
		if endpoint == nil {
			t.Fatalf("expected endpoint for net-1, got %#v", endpoints)
		}
		if endpoint.MacAddress != "02:42:ac:11:00:aa" {
			t.Errorf("unexpected MAC address: %q", endpoint.MacAddress)
		}
		if !reflect.DeepEqual(endpoint.Aliases, []string{"alias-net-1"}) {
			t.Errorf("unexpected aliases: %#v", endpoint.Aliases)
		}
		if endpoint.IPAMConfig == nil || endpoint.IPAMConfig.IPv4Address != "10.0.0.2" {
			t.Errorf("unexpected IPAM config: %#v", endpoint.IPAMConfig)
		}
		if !reflect.DeepEqual(endpoint.DriverOpts, map[string]string{"opt": "value"}) {
			t.Errorf("unexpected driver opts: %#v", endpoint.DriverOpts)
		}
		if endpoint.GwPriority != 10 {
			t.Errorf("unexpected gateway priority: %d", endpoint.GwPriority)
		}
		if endpoints["net-2"].MacAddress != "" {
			t.Errorf("unexpected MAC address for net-2: %q", endpoints["net-2"].MacAddress)
		}
	})

	t.Run("user_defined_mode_is_kept", func(t *testing.T) {
		endpoints, err := expandContainerCreateEndpointsConfig("mode-net", networks)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if len(endpoints) != 3 {
			t.Fatalf("expected 3 endpoints, got %d: %#v", len(endpoints), endpoints)
		}
		if endpoints["mode-net"] == nil {
			t.Fatalf("expected endpoint for the network_mode network, got %#v", endpoints)
		}
	})

	t.Run("user_defined_mode_already_declared", func(t *testing.T) {
		endpoints, err := expandContainerCreateEndpointsConfig("net-1", networks)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if len(endpoints) != 2 {
			t.Fatalf("expected 2 endpoints, got %d: %#v", len(endpoints), endpoints)
		}
		if endpoints["net-1"].MacAddress != "02:42:ac:11:00:aa" {
			t.Errorf("networks_advanced settings must not be overridden, got MAC %q", endpoints["net-1"].MacAddress)
		}
	})

	t.Run("duplicate_network", func(t *testing.T) {
		duplicate := rawNetwork("net-1", "02:42:ac:11:00:bb")
		if _, err := expandContainerCreateEndpointsConfig("bridge", append(networks, duplicate)); err == nil {
			t.Fatal("expected an error for a network declared twice")
		}
	})
}

func TestContainerNetworksAdvancedHasMacAddress(t *testing.T) {
	if containerNetworksAdvancedHasMacAddress([]interface{}{map[string]interface{}{"name": "a", "mac_address": ""}}) {
		t.Error("expected false when no MAC address is set")
	}
	if !containerNetworksAdvancedHasMacAddress([]interface{}{
		map[string]interface{}{"name": "a", "mac_address": ""},
		map[string]interface{}{"name": "b", "mac_address": "02:42:ac:11:00:aa"},
	}) {
		t.Error("expected true when a MAC address is set")
	}
}
