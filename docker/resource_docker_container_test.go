package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestMapTypeMapValsToStringSlice(t *testing.T) {
	typeMap := make(map[string]interface{})
	typeMap["foo"] = "bar"
	typeMap[""] = ""
	stringSlice := mapTypeMapValsToStringSlice(typeMap)
	if len(stringSlice) != 1 {
		t.Fatalf("slice should have length 1 but has %v", len(stringSlice))
	}
}

func TestAccDockerContainer_private_image(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	wd, _ := os.Getwd()
	dockerConfig := wd + "/../scripts/testing/dockerconfig.json"

	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDockerContainerPrivateImage, registry, dockerConfig, image),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
				),
			},
		},
		CheckDestroy: checkAndRemoveImages,
	})
}

func TestAccDockerContainer_basic(t *testing.T) {
	resourceName := "docker_container.foo"
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(resourceName, &c),
				),
			},
			// TODO mavogel: Will be done in #219
			// {
			// 	ResourceName:      resourceName,
			// 	ImportState:       true,
			// 	ImportStateVerify: true,
			// 	ImportStateVerifyIgnore: []string{
			// 		"attach",
			// 		"log_driver",
			// 		"logs",
			// 		"must_run",
			// 		"restart",
			// 		"rm",
			// 		"start",
			// 	},
			// },
		},
	})
}
func TestAccDockerContainer_basic_network(t *testing.T) {
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerWith2BridgeNetworkConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "bridge", ""),
					resource.TestCheckResourceAttrSet("docker_container.foo", "ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "gateway"),
					resource.TestCheckResourceAttr("docker_container.foo", "network_data.#", "2"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.gateway"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.gateway"),
				),
			},
		},
	})
}

func TestAccDockerContainer_2networks_withmode(t *testing.T) {
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainer2NetworksConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "bridge", ""),
					resource.TestCheckResourceAttrSet("docker_container.foo", "ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "gateway"),
					resource.TestCheckResourceAttr("docker_container.foo", "network_data.#", "2"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.gateway"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.gateway"),
					resource.TestCheckResourceAttr("docker_container.bar", "network_alias.#", "1"),
					resource.TestCheckResourceAttr("docker_container.bar", "bridge", ""),
					resource.TestCheckResourceAttrSet("docker_container.bar", "ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.bar", "ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.bar", "gateway"),
					resource.TestCheckResourceAttr("docker_container.bar", "network_data.#", "1"),
					resource.TestCheckResourceAttrSet("docker_container.bar", "network_data.0.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.bar", "network_data.0.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.bar", "network_data.0.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.bar", "network_data.0.gateway"),
				),
			},
		},
	})
}

func TestAccDockerContainerPath_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "/var/log", ErrCount: 0},
		{Value: "/tmp", ErrCount: 0},
		{Value: "C:\\Windows\\System32", ErrCount: 0},
		{Value: "C:\\Program Files\\MSBuild", ErrCount: 0},
		{Value: "test", ErrCount: 1},
		{Value: "C:Test", ErrCount: 1},
		{Value: "", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateDockerContainerPath(tc.Value, "docker_container")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Docker Container Path to trigger a validation error")
		}
	}
}

func TestAccDockerContainer_volume(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.Mounts) != 1 {
			return fmt.Errorf("Incorrect number of mounts: expected 1, got %d", len(c.Mounts))
		}

		for _, v := range c.Mounts {
			if v.Name != "testAccDockerContainerVolume_volume" {
				continue
			}

			if v.Destination != "/tmp/volume" {
				return fmt.Errorf("Bad destination on mount: expected /tmp/volume, got %q", v.Destination)
			}

			if v.Mode != "rw" {
				return fmt.Errorf("Bad mode on mount: expected rw, got %q", v.Mode)
			}

			return nil
		}

		return fmt.Errorf("Mount for testAccDockerContainerVolume_volume not found")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerVolumeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_mounts(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.Mounts) != 2 {
			return fmt.Errorf("Incorrect number of mounts: expected 2, got %d", len(c.Mounts))
		}

		for _, v := range c.Mounts {
			if v.Destination != "/mount/test" && v.Destination != "/mount/tmpfs" {
				return fmt.Errorf("Bad destination on mount: expected /mount/test or /mount/tmpfs, got %q", v.Destination)
			}
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerMountsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo_mounts", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_tmpfs(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.HostConfig.Tmpfs) != 1 {
			return fmt.Errorf("Incorrect number of tmpfs: expected 1, got %d", len(c.HostConfig.Tmpfs))
		}

		for mountPath, _ := range c.HostConfig.Tmpfs {
			if mountPath != "/mount/tmpfs" {
				return fmt.Errorf("Bad destination on tmpfs: expected /mount/tmpfs, got %q", mountPath)
			}
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerTmpfsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_sysctls(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.HostConfig.Sysctls) != 1 {
			return fmt.Errorf("Incorrect number of sysctls: expected 1, got %d", len(c.HostConfig.Sysctls))
		}

		if ctl, ok := c.HostConfig.Sysctls["net.ipv4.ip_forward"]; ok {
			if ctl != "1" {
				return fmt.Errorf("Bad value for sysctl net.ipv4.ip_forward: expected 1, got %s", ctl)
			}
		} else {
			return fmt.Errorf("net.ipv4.ip_forward not found in Sysctls")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerSysctlsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_groupadd_id(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.HostConfig.GroupAdd) != 1 || c.HostConfig.GroupAdd[0] != "100" {
			return fmt.Errorf("Wrong group add: %s", c.HostConfig.GroupAdd)
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerGroupAddIdConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_groupadd_name(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.HostConfig.GroupAdd) != 1 || c.HostConfig.GroupAdd[0] != "users" {
			return fmt.Errorf("Wrong group add: %s", c.HostConfig.GroupAdd)
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerGroupAddNameConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_groupadd_multiple(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.HostConfig.GroupAdd) != 3 {
			return fmt.Errorf("Wrong group add: %s", c.HostConfig.GroupAdd)
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerGroupAddMultipleConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_customized(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if len(c.Config.Entrypoint) < 3 ||
			(c.Config.Entrypoint[0] != "/bin/bash" &&
				c.Config.Entrypoint[1] != "-c" &&
				c.Config.Entrypoint[2] != "ping localhost") {
			return fmt.Errorf("Container wrong entrypoint: %s", c.Config.Entrypoint)
		}

		if c.Config.User != "root:root" {
			return fmt.Errorf("Container wrong user: %s", c.Config.User)
		}

		if c.HostConfig.RestartPolicy.Name == "on-failure" {
			if c.HostConfig.RestartPolicy.MaximumRetryCount != 5 {
				return fmt.Errorf("Container has wrong restart policy max retry count: %d", c.HostConfig.RestartPolicy.MaximumRetryCount)
			}
		} else {
			return fmt.Errorf("Container has wrong restart policy: %s", c.HostConfig.RestartPolicy.Name)
		}

		if c.HostConfig.Memory != (512 * 1024 * 1024) {
			return fmt.Errorf("Container has wrong memory setting: %d", c.HostConfig.Memory)
		}

		if c.HostConfig.MemorySwap != (2048 * 1024 * 1024) {
			return fmt.Errorf("Container has wrong memory swap setting: %d\n\r\tPlease check that you machine supports memory swap (you can do that by running 'docker info' command).", c.HostConfig.MemorySwap)
		}

		if c.HostConfig.ShmSize != (128 * 1024 * 1024) {
			return fmt.Errorf("Container has wrong shared memory setting: %d", c.HostConfig.ShmSize)
		}

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
		}

		if c.HostConfig.CpusetCpus != "0-1" {
			return fmt.Errorf("Container has wrong cpu set setting: %s", c.HostConfig.CpusetCpus)
		}

		if len(c.HostConfig.DNS) != 1 {
			return fmt.Errorf("Container does not have the correct number of dns entries: %d", len(c.HostConfig.DNS))
		}

		if c.HostConfig.DNS[0] != "8.8.8.8" {
			return fmt.Errorf("Container has wrong dns setting: %v", c.HostConfig.DNS[0])
		}

		if len(c.HostConfig.DNSOptions) != 1 {
			return fmt.Errorf("Container does not have the correct number of dns option entries: %d", len(c.HostConfig.DNS))
		}

		if c.HostConfig.DNSOptions[0] != "rotate" {
			return fmt.Errorf("Container has wrong dns option setting: %v", c.HostConfig.DNS[0])
		}

		if len(c.HostConfig.DNSSearch) != 1 {
			return fmt.Errorf("Container does not have the correct number of dns search entries: %d", len(c.HostConfig.DNS))
		}

		if c.HostConfig.DNSSearch[0] != "example.com" {
			return fmt.Errorf("Container has wrong dns search setting: %v", c.HostConfig.DNS[0])
		}

		if len(c.HostConfig.CapAdd) != 1 {
			return fmt.Errorf("Container does not have the correct number of Capabilities in ADD: %d", len(c.HostConfig.CapAdd))
		}

		if c.HostConfig.CapAdd[0] != "ALL" {
			return fmt.Errorf("Container has wrong CapAdd setting: %v", c.HostConfig.CapAdd[0])
		}

		if len(c.HostConfig.CapDrop) != 1 {
			return fmt.Errorf("Container does not have the correct number of Capabilities in Drop: %d", len(c.HostConfig.CapDrop))
		}

		if c.HostConfig.CapDrop[0] != "SYS_ADMIN" {
			return fmt.Errorf("Container has wrong CapDrop setting: %v", c.HostConfig.CapDrop[0])
		}

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
		}

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
		}

		if c.Config.Labels["env"] != "prod" || c.Config.Labels["role"] != "test" {
			return fmt.Errorf("Container does not have the correct labels")
		}

		if c.HostConfig.LogConfig.Type != "json-file" {
			return fmt.Errorf("Container does not have the correct log config: %s", c.HostConfig.LogConfig.Type)
		}

		if c.HostConfig.LogConfig.Config["max-size"] != "10m" {
			return fmt.Errorf("Container does not have the correct max-size log option: %v", c.HostConfig.LogConfig.Config["max-size"])
		}

		if c.HostConfig.LogConfig.Config["max-file"] != "20" {
			return fmt.Errorf("Container does not have the correct max-file log option: %v", c.HostConfig.LogConfig.Config["max-file"])
		}

		if len(c.HostConfig.ExtraHosts) != 2 {
			return fmt.Errorf("Container does not have correct number of extra host entries, got %d", len(c.HostConfig.ExtraHosts))
		}

		if c.HostConfig.ExtraHosts[0] != "testhost:10.0.1.0" {
			return fmt.Errorf("Container has incorrect extra host string at 0: %q", c.HostConfig.ExtraHosts[0])
		}

		if c.HostConfig.ExtraHosts[1] != "testhost2:10.0.2.0" {
			return fmt.Errorf("Container has incorrect extra host string at 1: %q", c.HostConfig.ExtraHosts[1])
		}

		if _, ok := c.NetworkSettings.Networks["test"]; !ok {
			return fmt.Errorf("Container is not connected to the right user defined network: test")
		}

		if len(c.HostConfig.Ulimits) != 2 {
			return fmt.Errorf("Container doesn't have 2 ulimits")
		}

		if c.HostConfig.Ulimits[1].Name != "nproc" {
			return fmt.Errorf("Container doesn't have a nproc ulimit")
		}

		if c.HostConfig.Ulimits[1].Hard != 1024 {
			return fmt.Errorf("Container doesn't have a correct nproc hard limit")
		}

		if c.HostConfig.Ulimits[1].Soft != 512 {
			return fmt.Errorf("Container doesn't have a correct mem nproc limit")
		}

		if c.HostConfig.Ulimits[0].Name != "nofile" {
			return fmt.Errorf("Container doesn't have a nofile ulimit")
		}

		if c.HostConfig.Ulimits[0].Hard != 262144 {
			return fmt.Errorf("Container doesn't have a correct nofile hard limit")
		}

		if c.HostConfig.Ulimits[0].Soft != 200000 {
			return fmt.Errorf("Container doesn't have a correct nofile soft limit")
		}

		if c.HostConfig.PidMode != "host" {
			return fmt.Errorf("Container doesn't have a correct pid mode")
		}
		if c.HostConfig.UsernsMode != "testuser:231072:65536" {
			return fmt.Errorf("Container doesn't have a correct userns mode")
		}
		if c.Config.WorkingDir != "/tmp" {
			return fmt.Errorf("Container doesn't have a correct working dir")
		}

		if c.HostConfig.IpcMode != "private" {
			return fmt.Errorf("Container doesn't have a correct ipc mode")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t); testAccCheckSwapLimit(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerCustomizedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					testCheckLabelMap("docker_container.foo", "labels", map[string]string{"env": "prod", "role": "test"}),
				),
			},
		},
	})
}

func testAccCheckSwapLimit(t *testing.T) {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	info, err := client.Info(context.Background())
	if err != nil {
		t.Fatalf("Failed to check swap limit capability: %s", err)
	}

	if !info.SwapLimit {
		t.Skip("Swap limit capability not available, skipping test")
	}
}

func TestAccDockerContainer_upload(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient

		srcPath := "/terraform/test.txt"
		r, _, err := client.CopyFromContainer(context.Background(), c.ID, srcPath)
		if err != nil {
			return fmt.Errorf("Unable to download a file from container: %s", err)
		}

		tr := tar.NewReader(r)
		if header, err := tr.Next(); err != nil {
			return fmt.Errorf("Unable to read content of tar archive: %s", err)
		} else {
			mode := strconv.FormatInt(header.Mode, 8)
			if !strings.HasSuffix(mode, "744") {
				return fmt.Errorf("File permissions are incorrect: %s", mode)
			}
		}

		fbuf := new(bytes.Buffer)
		fbuf.ReadFrom(tr)
		content := fbuf.String()

		if content != "foo" {
			return fmt.Errorf("file content is invalid")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerUploadConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "1"),
					// NOTE mavogel: current the terraform-plugin-sdk it's likely that
					// the acceptance testing framework shims (still using the older flatmap-style addressing)
					// are missing a conversion with the hashes.
					// See https://github.com/hashicorp/terraform-plugin-sdk/issues/196
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", "foo"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", ""),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test.txt"),
				),
			},
		},
	})
}

func TestAccDockerContainer_uploadAsBase64(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(srcPath, wantedContent, filePerm string) func(*terraform.State) error {
		return func(*terraform.State) error {
			client := testAccProvider.Meta().(*ProviderConfig).DockerClient

			r, _, err := client.CopyFromContainer(context.Background(), c.ID, srcPath)
			if err != nil {
				return fmt.Errorf("Unable to download a file from container: %s", err)
			}

			tr := tar.NewReader(r)
			if header, err := tr.Next(); err != nil {
				return fmt.Errorf("Unable to read content of tar archive: %s", err)
			} else {
				mode := strconv.FormatInt(header.Mode, 8)
				if !strings.HasSuffix(mode, filePerm) {
					return fmt.Errorf("File permissions are incorrect: %s", mode)
				}
			}

			fbuf := new(bytes.Buffer)
			fbuf.ReadFrom(tr)
			gotContent := fbuf.String()

			if wantedContent != gotContent {
				return fmt.Errorf("file content is invalid: want: %q, got: %q", wantedContent, gotContent)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerUploadBase64Config,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck("/terraform/test1.txt", "894fc3f56edf2d3a4c5fb5cb71df910f958a2ed8", "744"),
					testCheck("/terraform/test2.txt", "foobar", "100644"),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "2"),
					// NOTE: see comment above
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", ""),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", "ODk0ZmMzZjU2ZWRmMmQzYTRjNWZiNWNiNzFkZjkxMGY5NThhMmVkOA=="),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test1.txt"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content", "foo"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content_base64", ""),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.executable", "false"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.file", "/terraform/test2.txt"),
				),
			},
			// We add a second on purpose to detect if there is a dirty plan
			// although the file content did not change
			{
				Config: testAccDockerContainerUploadBase64Config,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck("/terraform/test1.txt", "894fc3f56edf2d3a4c5fb5cb71df910f958a2ed8", "744"),
					testCheck("/terraform/test2.txt", "foobar", "100644"),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "2"),
					// NOTE: see comment above
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", ""),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", "ODk0ZmMzZjU2ZWRmMmQzYTRjNWZiNWNiNzFkZjkxMGY5NThhMmVkOA=="),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test1.txt"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content", "foo"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content_base64", ""),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.executable", "false"),
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.file", "/terraform/test2.txt"),
				),
			},
		},
	})
}

func TestAccDockerContainer_multipleUploadContentsConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_image" "foo" {
					name         = "nginx:latest"
					keep_locally = true
				}
				
				resource "docker_container" "foo" {
					name     = "tf-test"
					image    = "${docker_image.foo.latest}"
					must_run = "false"
				
					upload {
						content        = "foobar"
						content_base64 = "${base64encode("barbaz")}"
						file           = "/terraform/test1.txt"
						executable     = true
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*only one of 'content' or 'content_base64' can be specified.*`),
			},
		},
	})
}

func TestAccDockerContainer_noUploadContentsConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_image" "foo" {
					name         = "nginx:latest"
					keep_locally = true
				}
				
				resource "docker_container" "foo" {
					name     = "tf-test"
					image    = "${docker_image.foo.latest}"
					must_run = "false"
				
					upload {
						file           = "/terraform/test1.txt"
						executable     = true
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.* neither 'content', nor 'content_base64' was set.*`),
			},
		},
	})
}

func TestAccDockerContainer_device(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient

		createExecOpts := types.ExecConfig{
			Cmd: []string{"dd", "if=/dev/zero_test", "of=/tmp/test.txt", "count=10", "bs=1"},
		}

		exec, err := client.ContainerExecCreate(context.Background(), c.ID, createExecOpts)
		if err != nil {
			return fmt.Errorf("Unable to create a exec instance on container: %s", err)
		}

		startExecOpts := types.ExecStartCheck{}
		if err := client.ContainerExecStart(context.Background(), exec.ID, startExecOpts); err != nil {
			return fmt.Errorf("Unable to run exec a instance on container: %s", err)
		}

		srcPath := "/tmp/test.txt"
		out, _, err := client.CopyFromContainer(context.Background(), c.ID, srcPath)
		if err != nil {
			return fmt.Errorf("Unable to download a file from container: %s", err)
		}

		tr := tar.NewReader(out)
		if _, err := tr.Next(); err != nil {
			return fmt.Errorf("Unable to read content of tar archive: %s", err)
		}

		fbuf := new(bytes.Buffer)
		fbuf.ReadFrom(tr)
		content := fbuf.Bytes()

		if len(content) != 10 {
			return fmt.Errorf("Incorrect size of file: %d", len(content))
		}
		for _, value := range content {
			if value != 0 {
				return fmt.Errorf("Incorrect content in file: %v", content)
			}
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerDeviceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_port_internal(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		portMap := c.NetworkSettings.NetworkSettingsBase.Ports
		portBindings, ok := portMap["80/tcp"]
		if !ok || len(portMap["80/tcp"]) == 0 {
			return fmt.Errorf("Port 80 on tcp is not set")
		}

		portBindingsLength := len(portBindings)
		if portBindingsLength != 1 {
			return fmt.Errorf("Expected 1 binding on port 80, but was %d", portBindingsLength)
		}

		if len(portBindings[0].HostIP) == 0 {
			return fmt.Errorf("Expected host IP to be set, but was empty")
		}

		if len(portBindings[0].HostPort) == 0 {
			return fmt.Errorf("Expected host port to be set, but was empty")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerInternalPortConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.internal", "80"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.ip", "0.0.0.0"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.protocol", "tcp"),
					testValueHigherEqualThan("docker_container.foo", "ports.0.external", 32768),
				),
			},
		},
	})
}

func TestAccDockerContainer_port_multiple_internal(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		portMap := c.NetworkSettings.NetworkSettingsBase.Ports
		portBindings, ok := portMap["80/tcp"]
		if !ok || len(portMap["80/tcp"]) == 0 {
			return fmt.Errorf("Port 80 on tcp is not set")
		}

		portBindingsLength := len(portBindings)
		if portBindingsLength != 1 {
			return fmt.Errorf("Expected 1 binding on port 80, but was %d", portBindingsLength)
		}

		if len(portBindings[0].HostIP) == 0 {
			return fmt.Errorf("Expected host IP to be set, but was empty")
		}

		if len(portBindings[0].HostPort) == 0 {
			return fmt.Errorf("Expected host port to be set, but was empty")
		}

		portBindings, ok = portMap["81/tcp"]
		if !ok || len(portMap["81/tcp"]) == 0 {
			return fmt.Errorf("Port 81 on tcp is not set")
		}

		portBindingsLength = len(portBindings)
		if portBindingsLength != 1 {
			return fmt.Errorf("Expected 1 binding on port 81, but was %d", portBindingsLength)
		}

		if len(portBindings[0].HostIP) == 0 {
			return fmt.Errorf("Expected host IP to be set, but was empty")
		}

		if len(portBindings[0].HostPort) == 0 {
			return fmt.Errorf("Expected host port to be set, but was empty")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerMultipleInternalPortConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.internal", "80"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.ip", "0.0.0.0"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.protocol", "tcp"),
					testValueHigherEqualThan("docker_container.foo", "ports.0.external", 32768),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.internal", "81"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.ip", "0.0.0.0"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.protocol", "tcp"),
					testValueHigherEqualThan("docker_container.foo", "ports.1.external", 32768),
				),
			},
		},
	})
}

func TestAccDockerContainer_port(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		portMap := c.NetworkSettings.NetworkSettingsBase.Ports
		portBindings, ok := portMap["80/tcp"]
		if !ok || len(portMap["80/tcp"]) == 0 {
			return fmt.Errorf("Port 80 on tcp is not set")
		}

		portBindingsLength := len(portBindings)
		if portBindingsLength != 1 {
			return fmt.Errorf("Expected 1 binding on port 80, but was %d", portBindingsLength)
		}

		if len(portBindings[0].HostIP) == 0 {
			return fmt.Errorf("Expected host IP to be set, but was empty")
		}

		if len(portBindings[0].HostPort) == 0 {
			return fmt.Errorf("Expected host port to be set, but was empty")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerPortConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.#", "1"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.internal", "80"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.ip", "0.0.0.0"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.external", "32787"),
				),
			},
			{
				Config: testAccDockerContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.#", "0"),
				),
			},
		},
	})
}

func TestAccDockerContainer_multiple_ports(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		portMap := c.NetworkSettings.NetworkSettingsBase.Ports
		portBindings, ok := portMap["80/tcp"]
		if !ok || len(portMap["80/tcp"]) == 0 {
			return fmt.Errorf("Port 80 on tcp is not set")
		}

		portBindingsLength := len(portBindings)
		if portBindingsLength != 1 {
			return fmt.Errorf("Expected 1 binding on port 80, but was %d", portBindingsLength)
		}

		if len(portBindings[0].HostIP) == 0 {
			return fmt.Errorf("Expected host IP to be set, but was empty")
		}

		if len(portBindings[0].HostPort) == 0 {
			return fmt.Errorf("Expected host port to be set, but was empty")
		}

		portBindings, ok = portMap["81/tcp"]
		if !ok || len(portMap["81/tcp"]) == 0 {
			return fmt.Errorf("Port 81 on tcp is not set")
		}

		portBindingsLength = len(portBindings)
		if portBindingsLength != 1 {
			return fmt.Errorf("Expected 1 binding on port 81, but was %d", portBindingsLength)
		}

		if len(portBindings[0].HostIP) == 0 {
			return fmt.Errorf("Expected host IP to be set, but was empty")
		}

		if len(portBindings[0].HostPort) == 0 {
			return fmt.Errorf("Expected host port to be set, but was empty")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerMultiplePortConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.#", "2"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.internal", "80"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.ip", "0.0.0.0"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.protocol", "tcp"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.0.external", "32787"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.internal", "81"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.ip", "0.0.0.0"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.protocol", "tcp"),
					resource.TestCheckResourceAttr("docker_container.foo", "ports.1.external", "32788"),
				),
			},
		},
	})
}

func TestAccDockerContainer_rm(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if !c.HostConfig.AutoRemove {
			return fmt.Errorf("Container doesn't have a correct autoremove flag")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccContainerWaitConditionRemoved("docker_container.foo", &c),
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerRmConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "rm", "true"),
				),
			},
		},
	})
}

func TestAccDockerContainer_readonly(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if !c.HostConfig.ReadonlyRootfs {
			return fmt.Errorf("Container isn't readonly")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerReadOnlyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "read_only", "true"),
				),
			},
		},
	})
}

func TestAccDockerContainer_healthcheck(t *testing.T) {
	var c types.ContainerJSON
	testCheck := func(*terraform.State) error {
		if !reflect.DeepEqual(c.Config.Healthcheck.Test, []string{"CMD", "/bin/true"}) {
			return fmt.Errorf("Container doesn't have a correct healthcheck test")
		}
		if c.Config.Healthcheck.Interval != 30000000000 {
			return fmt.Errorf("Container doesn't have a correct healthcheck interval")
		}
		if c.Config.Healthcheck.Timeout != 5000000000 {
			return fmt.Errorf("Container doesn't have a correct healthcheck timeout")
		}
		if c.Config.Healthcheck.StartPeriod != 15000000000 {
			return fmt.Errorf("Container doesn't have a correct healthcheck retries")
		}
		if c.Config.Healthcheck.Retries != 10 {
			return fmt.Errorf("Container doesn't have a correct healthcheck retries")
		}
		return nil
	}
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerHealthcheckConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_nostart(t *testing.T) {
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerNoStartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerNotRunning("docker_container.foo", &c),
				),
			},
		},
	})
}

func TestAccDockerContainer_attach(t *testing.T) {
	var c types.ContainerJSON

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerAttachConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerNotRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "attach", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "must_run", "false"),
				),
			},
		},
	})
}

func TestAccDockerContainer_logs(t *testing.T) {
	var c types.ContainerJSON

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerLogsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerNotRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "attach", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "logs", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "must_run", "false"),
					resource.TestCheckResourceAttr("docker_container.foo", "container_logs", "\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00021\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00022\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00023\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00024\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00025\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00026\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00027\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00028\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u00029\n\u0001\u0000\u0000\u0000\u0000\u0000\u0000\u000310\n"),
				),
			},
		},
	})
}

func TestAccDockerContainer_exitcode(t *testing.T) {
	var c types.ContainerJSON

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerExitCodeConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerWaitConditionNotRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "exit_code", "123"),
				),
			},
		},
	})
}

func TestAccDockerContainer_ipv4address(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		networks := c.NetworkSettings.Networks

		if len(networks) != 1 {
			return fmt.Errorf("Container doesn't have a correct network")
		}
		if _, ok := networks["tf-test"]; !ok {
			return fmt.Errorf("Container doesn't have a correct network")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig == nil {
			return fmt.Errorf("Container doesn't have a correct IPAM config")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig.IPv4Address != "10.0.1.123" {
			return fmt.Errorf("Container doesn't have a correct IPv4 address")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerNetworksIPv4AddressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
				),
			},
		},
	})
}

func TestAccDockerContainer_ipv6address(t *testing.T) {
	t.Skip("mavogel: need to fix ipv6 network state")
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		networks := c.NetworkSettings.Networks

		if len(networks) != 1 {
			return fmt.Errorf("Container doesn't have a correct network")
		}
		if _, ok := networks["tf-test"]; !ok {
			return fmt.Errorf("Container doesn't have a correct network")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig == nil {
			return fmt.Errorf("Container doesn't have a correct IPAM config")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig.IPv6Address != "fd00:0:0:0::123" {
			return fmt.Errorf("Container doesn't have a correct IPv6 address")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerNetworksIPv6AddressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
				),
			},
		},
	})
}

func TestAccDockerContainer_dualstackaddress(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		networks := c.NetworkSettings.Networks

		if len(networks) != 1 {
			return fmt.Errorf("Container doesn't have a correct network")
		}
		if _, ok := networks["tf-test"]; !ok {
			return fmt.Errorf("Container doesn't have a correct network")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig == nil {
			return fmt.Errorf("Container doesn't have a correct IPAM config")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig.IPv4Address != "10.0.1.123" {
			return fmt.Errorf("Container doesn't have a correct IPv4 address")
		}
		if c.NetworkSettings.Networks["tf-test"].IPAMConfig.IPv6Address != "fd00:0:0:0::123" {
			return fmt.Errorf("Container doesn't have a correct IPv6 address")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerContainerNetworksDualStackAddressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
				),
			},
		},
	})
}

func testAccContainerRunning(n string, container *types.ContainerJSON) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				inspected, err := client.ContainerInspect(context.Background(), c.ID)
				if err != nil {
					return fmt.Errorf("Container could not be inspected: %s", err)
				}
				*container = inspected
				return nil
			}
		}

		return fmt.Errorf("Container not found: %s", rs.Primary.ID)
	}
}

func testAccContainerNotRunning(n string, container *types.ContainerJSON) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{
			All: true,
		})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				inspected, err := client.ContainerInspect(context.Background(), c.ID)
				if err != nil {
					return fmt.Errorf("Container could not be inspected: %s", err)
				}
				*container = inspected

				if container.State.Running {
					return fmt.Errorf("Container is running: %s", rs.Primary.ID)
				}
			}
		}

		return nil
	}
}

func testAccContainerWaitConditionNotRunning(n string, ct *types.ContainerJSON) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		statusC, errC := client.ContainerWait(ctx, rs.Primary.ID, container.WaitConditionNotRunning)

		select {
		case err := <-errC:
			{
				if err != nil {
					return fmt.Errorf("Container is still running")
				}
			}

		case <-statusC:
		}

		return nil
	}
}

func testAccContainerWaitConditionRemoved(n string, ct *types.ContainerJSON) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		statusC, errC := client.ContainerWait(ctx, rs.Primary.ID, container.WaitConditionRemoved)

		select {
		case err := <-errC:
			{
				if err != nil {
					return fmt.Errorf("Container has not been removed")
				}
			}

		case <-statusC:
		}

		return nil
	}
}

func testValueHigherEqualThan(name, key string, value int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		vRaw, ok := is.Attributes[key]
		if !ok {
			return fmt.Errorf("%s: Attribute '%s' not found", name, key)
		}

		v, err := strconv.Atoi(vRaw)
		if err != nil {
			return fmt.Errorf("'%s' is not a number", vRaw)
		}

		if v < value {
			return fmt.Errorf("'%v' is smaller than '%v', but was expected to be equal or greater", v, value)
		}

		return nil
	}
}

const testAccDockerContainerPrivateImage = `
provider "docker" {
	alias = "private"
	registry_auth {
		address = "%s"
		config_file = "%s"
	}
}

resource "docker_container" "foo" {
	provider = "docker.private"
	name  = "tf-test"
	image = "%s"
}
`

const testAccDockerContainerConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
}
`

const testAccDockerContainerWith2BridgeNetworkConfig = `
resource "docker_network" "tftest" {
  name = "tftest-contnw"
}

resource "docker_network" "tftest_2" {
  name = "tftest-contnw-2"
}

resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name  	 = "tf-test"
	image 	 = "${docker_image.foo.latest}"
	networks = [
		"${docker_network.tftest.name}",
		"${docker_network.tftest_2.name}"
	]
}
`

const testAccDockerContainerVolumeConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_volume" "foo" {
    name = "testAccDockerContainerVolume_volume"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

    volumes {
        volume_name = "${docker_volume.foo.name}"
        container_path = "/tmp/volume"
        read_only = false
    }
}
`

const testAccDockerContainerMountsConfig = `
resource "docker_image" "foo_mounts" {
	name = "nginx:latest"
}

resource "docker_volume" "foo_mounts" {
    name = "testAccDockerContainerMounts_volume"
}

resource "docker_container" "foo_mounts" {
	name = "tf-test"
	image = "${docker_image.foo_mounts.latest}"

	mounts {
		target      = "/mount/test"
		source      = "${docker_volume.foo_mounts.name}"
		type        = "volume"
		read_only   = true
	}
	mounts {
		target  = "/mount/tmpfs"
		type    = "tmpfs"
	}
}
`

const testAccDockerContainerTmpfsConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	tmpfs = {
		"/mount/tmpfs" = "rw,noexec,nosuid"
	}
}
`

const testAccDockerContainerCustomizedConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	entrypoint = ["/bin/bash", "-c", "ping localhost"]
	user = "root:root"
	restart = "on-failure"
	destroy_grace_seconds = 10
	max_retry_count = 5
	memory = 512
	shm_size = 128
	memory_swap = 2048
	cpu_shares = 32
	cpu_set = "0-1"

	capabilities {
		add  = ["ALL"]
		drop = ["SYS_ADMIN"]
	}

	dns = ["8.8.8.8"]
	dns_opts = ["rotate"]
	dns_search = ["example.com"]
	labels {
		label = "env"
		value = "prod"
	}
	labels {
		label = "role"
		value = "test"
	}
	log_driver = "json-file"
	log_opts = {
		max-size = "10m"
		max-file = 20
	}
	network_mode = "bridge"

	networks_advanced {
		name = "${docker_network.test_network.name}"
		aliases = ["tftest"]
	}

	host {
		host = "testhost"
		ip = "10.0.1.0"
	}

	host {
		host = "testhost2"
		ip = "10.0.2.0"
	}

	ulimit {
		name = "nproc"
		hard = 1024
		soft = 512
	}

	ulimit {
		name = "nofile"
		hard = 262144
		soft = 200000
	}

	pid_mode 		= "host"
	userns_mode = "testuser:231072:65536"
	ipc_mode = "private"
	working_dir = "/tmp"
}

resource "docker_network" "test_network" {
  name = "test"
}
`

const testAccDockerContainerUploadConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	upload {
		content = "foo"
		file = "/terraform/test.txt"
		executable = true
	}
}
`

const testAccDockerContainerUploadBase64Config = `
resource "docker_image" "foo" {
	name         = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name  = "tf-test"
	image = "${docker_image.foo.latest}"

	upload {
		content_base64 = "${base64encode("894fc3f56edf2d3a4c5fb5cb71df910f958a2ed8")}"
		file           = "/terraform/test1.txt"
		executable     = true
	}

	upload {
		content = "foobar"
		file    = "/terraform/test2.txt"
	}
}
`

const testAccDockerContainerDeviceConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	devices {
    	host_path = "/dev/zero"
    	container_path = "/dev/zero_test"
	}
}
`

const testAccDockerContainerInternalPortConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	ports {
		internal = 80
	}
}
`

const testAccDockerContainerMultipleInternalPortConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	ports {
		internal = 80
	}

	ports {
		internal = 81
	}
}
`
const testAccDockerContainerPortConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	ports {
		internal = 80
		external = 32787
	}
}
`
const testAccDockerContainerMultiplePortConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	ports {
		internal = 80
		external = 32787
	}

	ports {
		internal = 81
		external = 32788
	}

}
`

const testAccDockerContainer2NetworksConfig = `
resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_network" "test_network_1" {
  name = "tftest-1"
}

resource "docker_network" "test_network_2" {
  name = "tftest-2"
}

resource "docker_container" "foo" {
  name          = "tf-test"
  image         = "${docker_image.foo.latest}"
  network_mode  = "${docker_network.test_network_1.name}"
  networks      = ["${docker_network.test_network_2.name}"]
  network_alias = ["tftest-container"]
}

resource "docker_container" "bar" {
  name          = "tf-test-bar"
  image         = "${docker_image.foo.latest}"
  network_mode  = "bridge"
  networks      = ["${docker_network.test_network_2.name}"]
  network_alias = ["tftest-container-foo"]
}
`

const testAccDockerContainerHealthcheckConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
  name  = "tf-test"
  image = "${docker_image.foo.latest}"

  healthcheck {
    test         = ["CMD", "/bin/true"]
    interval     = "30s"
    timeout      = "5s"
    start_period = "15s"
    retries      = 10
  }
}
`
const testAccDockerContainerNoStartConfig = `
resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name     = "tf-test"
  image    = "nginx:latest"
  start    = false
  must_run = false
}
`
const testAccDockerContainerNetworksIPv4AddressConfig = `
resource "docker_network" "test" {
	name = "tf-test"
	ipam_config {
		subnet = "10.0.1.0/24"
	}
}
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}
resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	networks_advanced {
		name = "${docker_network.test.name}"
		ipv4_address = "10.0.1.123"
	}
}
`
const testAccDockerContainerNetworksIPv6AddressConfig = `
resource "docker_network" "test" {
	name = "tf-test"
	ipv6 = true
	ipam_config {
		subnet = "fd00::1/64"
	}
}
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}
resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	networks_advanced {
		name = "${docker_network.test.name}"
		ipv6_address = "fd00:0:0:0::123"
	}
}
`
const testAccDockerContainerNetworksDualStackAddressConfig = `
resource "docker_network" "test" {
	name = "tf-test"
	ipv6 = true

	ipam_config {
		subnet = "10.0.1.0/24"
	}

	ipam_config {
		subnet = "fd00::1/64"
	}
}
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}
resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	networks_advanced {
		name = "${docker_network.test.name}"
		ipv4_address = "10.0.1.123"
		ipv6_address = "fd00:0:0:0::123"
	}
}
`
const testAccDockerContainerRmConfig = `
resource "docker_image" "foo" {
	name = "busybox:latest"
	keep_locally = true
}
 resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	command = ["/bin/sleep", "15"]
	rm = true
}
`
const testAccDockerContainerReadOnlyConfig = `
resource "docker_image" "foo" {
	name = "busybox:latest"
	keep_locally = true
}
 resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	command = ["/bin/sleep", "15"]
	read_only = true
}
`
const testAccDockerContainerAttachConfig = `
resource "docker_image" "foo" {
	name = "busybox:latest"
	keep_locally = true
}
 resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
	command = ["/bin/sh", "-c", "for i in $(seq 1 15); do sleep 1; done"]
	attach = true
	must_run = false
}
`
const testAccDockerContainerLogsConfig = `
resource "docker_image" "foo" {
  name         = "busybox:latest"
  keep_locally = true
}

resource "docker_container" "foo" {
  name     = "tf-test"
  image    = "${docker_image.foo.latest}"
  command  = ["/bin/sh", "-c", "for i in $(seq 1 10); do echo \"$i\"; done"]
  attach   = true
  logs     = true
  must_run = false
}
`
const testAccDockerContainerExitCodeConfig = `
resource "docker_image" "foo" {
name = "busybox:latest"
keep_locally = true
}
 resource "docker_container" "foo" {
name = "tf-test"
image = "${docker_image.foo.latest}"
command = ["/bin/sh", "-c", "exit 123"]
attach = true
must_run = false
}
`

const testAccDockerContainerSysctlsConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	sysctls = {
		"net.ipv4.ip_forward" = "1"
	}
}
`

const testAccDockerContainerGroupAddNameConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	group_add = [
		"users"
	]
}
`

const testAccDockerContainerGroupAddIdConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	group_add = [
		100
	]
}
`

const testAccDockerContainerGroupAddMultipleConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"

	group_add = [
		1,
		2,
		3,
	]
}
`
