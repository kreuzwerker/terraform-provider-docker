package provider

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
	dockerConfig := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "dockerconfig.json"), "\\", "\\\\")
	ctx := context.Background()

	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerPrivateImage"), registry, dockerConfig, image),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerContainer_basic(t *testing.T) {
	resourceName := "docker_container.foo"
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(resourceName, &c),
				),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUpdateConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(resourceName, &c),
				),
			},
			{
				ResourceName:      "docker_container.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"attach",
					"log_driver",
					"logs",
					"must_run",
					"restart",
					"rm",
					"start",
					"wait",
					"wait_timeout",
					"container_logs",
					"destroy_grace_seconds",
					"upload",
					"remove_volumes",
					"init",

					// TODO mavogel: Will be done in #74 (import resources)
					"volumes",
					"network_advanced",
					"container_read_refresh_timeout_milliseconds",
				},
			},
		},
	})
}

func TestAccDockerContainer_init(t *testing.T) {
	resourceName := "docker_container.fooinit"
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerInitConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning(resourceName, &c),
				),
			},
			{
				ResourceName:      "docker_container.fooinit",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"attach",
					"log_driver",
					"logs",
					"must_run",
					"restart",
					"rm",
					"start",
					"wait",
					"wait_timeout",
					"container_logs",
					"destroy_grace_seconds",
					"upload",
					"remove_volumes",

					// TODO mavogel: Will be done in #74 (import resources)
					"volumes",
					"network_advanced",
					"container_read_refresh_timeout_milliseconds",
				},
			},
		},
	})
}

func TestAccDockerContainer_basic_network(t *testing.T) {
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerWith2BridgeNetworkConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "bridge", ""),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainer2NetworksConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					resource.TestCheckResourceAttr("docker_container.foo", "bridge", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "network_data.#", "2"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.gateway"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.network_name"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.ip_address"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.ip_prefix_length"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.1.gateway"),
					resource.TestCheckResourceAttr("docker_container.bar", "bridge", ""),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerVolumeConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerMountsConfig"),
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

		for mountPath := range c.HostConfig.Tmpfs {
			if mountPath != "/mount/tmpfs" {
				return fmt.Errorf("Bad destination on tmpfs: expected /mount/tmpfs, got %q", mountPath)
			}
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerTmpfsConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerSysctlsConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerGroupAddIdConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerGroupAddNameConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerGroupAddMultipleConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_tty(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if !c.Config.Tty {
			return fmt.Errorf("Tty not enabled")
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerTTYConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
}

func TestAccDockerContainer_STDIN_Enabled(t *testing.T) {
	var c types.ContainerJSON

	testCheck := func(*terraform.State) error {
		if !c.Config.OpenStdin {
			return fmt.Errorf("STDIN not enabled")
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerSTDIN_Config"),
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

		// Disabled for tests due to
		// --storage-opt is supported only for overlay over xfs with 'pquota' mount option
		// see https://github.com/kreuzwerker/terraform-provider-docker/issues/177
		// if c.HostConfig.StorageOpt["size"] != "100Mi" {
		// 	return fmt.Errorf("Container does not have the correct size storage option: %v", c.HostConfig.StorageOpt["size"])
		// }

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccCheckSwapLimit(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerCustomizedConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					testCheckLabelMap("docker_container.foo", "labels", map[string]string{"env": "prod", "role": "test", "maintainer": "NGINX Docker Maintainers <docker-maint@nginx.com>"}),
				),
			},
		},
	})
}

func testAccCheckSwapLimit(t *testing.T) {
	ctx := context.Background()
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient
	info, err := client.Info(ctx)
	if err != nil {
		t.Fatalf("Failed to check swap limit capability: %s", err)
	}

	if !info.SwapLimit {
		t.Skip("Swap limit capability not available, skipping test")
	}
}

func TestAccDockerContainer_upload(t *testing.T) {
	var c types.ContainerJSON
	ctx := context.Background()

	testCheck := func(*terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient

		srcPath := "/terraform/test.txt"
		r, _, err := client.CopyFromContainer(ctx, c.ID, srcPath)
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
		if _, err := fbuf.ReadFrom(tr); err != nil {
			return err
		}
		content := fbuf.String()

		if content != "foo" {
			return fmt.Errorf("file content is invalid")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUploadConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "1"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", "foo"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test.txt"),
				),
			},
		},
	})
}

func TestAccDockerContainer_uploadSource(t *testing.T) {
	var c types.ContainerJSON
	ctx := context.Background()

	wd, _ := os.Getwd()
	testFile := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "testingFile"), "\\", "\\\\")
	testFileContent, _ := ioutil.ReadFile(testFile)

	testCheck := func(*terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient

		srcPath := "/terraform/test.txt"
		r, _, err := client.CopyFromContainer(ctx, c.ID, srcPath)
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
		if _, err := fbuf.ReadFrom(tr); err != nil {
			return err
		}
		content := fbuf.String()
		if content != string(testFileContent) {
			return fmt.Errorf("file content is invalid")
		}

		// we directly exec the container and print the creation timestamp
		// which is easier to use the native docker sdk, by creating, running and attaching a reader to the command.
		execReponse, err := exec.Command("docker", "exec", "-t", "tf-test", "find", "/terraform", "-maxdepth", "1", "-name", "test.txt", "-printf", "%CY-%Cm-%Cd").Output()
		if err != nil {
			return fmt.Errorf("Unable to exec command: %s", err)
		}

		fileCreationTime, err := time.Parse("2006-01-02", string(execReponse))
		if err != nil {
			return fmt.Errorf("Unable to parse file creation time into format: %s", err)
		}

		if fileCreationTime.IsZero() {
			return fmt.Errorf("file creation time is zero: %s", err)
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUploadSourceConfig"), testFile),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "1"),
					// TODO mavogel: should be content of the source be written to this attribute?
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", "foo"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test.txt"),
				),
			},
		},
	})
}

func TestAccDockerContainer_uploadSourceHash(t *testing.T) {
	var c types.ContainerJSON
	var firstRunId string

	wd, _ := os.Getwd()
	testFile := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "testingFile"), "\\", "\\\\")
	hash, _ := ioutil.ReadFile(testFile + ".base64")
	grabFirstCheck := func(*terraform.State) error {
		firstRunId = c.ID
		return nil
	}
	testCheck := func(*terraform.State) error {
		if c.ID == firstRunId {
			return fmt.Errorf("Container should have been recreated due to changed hash")
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUploadSourceHashConfig"), testFile, string(hash)),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					grabFirstCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUploadSourceHashConfig"), testFile, string(hash)+"arbitrary"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "1"),
				),
			},
		},
	})
}

func TestAccDockerContainer_uploadAsBase64(t *testing.T) {
	var c types.ContainerJSON
	ctx := context.Background()

	testCheck := func(srcPath, wantedContent, filePerm string) func(*terraform.State) error {
		return func(*terraform.State) error {
			client := testAccProvider.Meta().(*ProviderConfig).DockerClient

			r, _, err := client.CopyFromContainer(ctx, c.ID, srcPath)
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
			if _, err := fbuf.ReadFrom(tr); err != nil {
				return err
			}
			gotContent := fbuf.String()

			if wantedContent != gotContent {
				return fmt.Errorf("file content is invalid: want: %q, got: %q", wantedContent, gotContent)
			}

			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUploadBase64Config"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					// DevSkim: ignore DS173237
					testCheck("/terraform/test1.txt", "894fc3f56edf2d3a4c5fb5cb71df910f958a2ed8", "744"),
					testCheck("/terraform/test2.txt", "foobar", "100644"),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "2"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", "ODk0ZmMzZjU2ZWRmMmQzYTRjNWZiNWNiNzFkZjkxMGY5NThhMmVkOA=="),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test1.txt"),
					// TODO mavogel: should be content of the source be written to this attribute?
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content", "foo"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content_base64", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.1.executable", "false"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.1.file", "/terraform/test2.txt"),
				),
			},
			// We add a second on purpose to detect if there is a dirty plan
			// although the file content did not change
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerUploadBase64Config"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					// DevSkim: ignore DS173237
					testCheck("/terraform/test1.txt", "894fc3f56edf2d3a4c5fb5cb71df910f958a2ed8", "744"),
					testCheck("/terraform/test2.txt", "foobar", "100644"),
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.#", "2"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.content_base64", "ODk0ZmMzZjU2ZWRmMmQzYTRjNWZiNWNiNzFkZjkxMGY5NThhMmVkOA=="),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.executable", "true"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.0.file", "/terraform/test1.txt"),
					// TODO mavogel: should be content of the source be written to this attribute?
					// resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content", "foo"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.1.content_base64", ""),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.1.executable", "false"),
					resource.TestCheckResourceAttr("docker_container.foo", "upload.1.file", "/terraform/test2.txt"),
				),
			},
		},
	})
}

func TestAccDockerContainer_multipleUploadContentsConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_image" "foo" {
					name         = "nginx:latest"
					keep_locally = true
				}

				resource "docker_container" "foo" {
					name     = "tf-test"
					image    = docker_image.foo.image_id
					must_run = "false"

					upload {
						content        = "foobar"
						content_base64 = base64encode("barbaz")
						file           = "/terraform/test1.txt"
						executable     = true
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.*only one of 'content', 'content_base64', or 'source' can be set.*`),
			},
		},
	})
}

func TestAccDockerContainer_noUploadContentsConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "docker_image" "foo" {
					name         = "nginx:latest"
					keep_locally = true
				}

				resource "docker_container" "foo" {
					name     = "tf-test"
					image    = docker_image.foo.image_id
					must_run = "false"

					upload {
						file           = "/terraform/test1.txt"
						executable     = true
					}
				}
				`,
				ExpectError: regexp.MustCompile(`.* one of 'content', 'content_base64', or 'source' must be set.*`),
			},
		},
	})
}

func TestAccDockerContainer_device(t *testing.T) {
	var c types.ContainerJSON
	ctx := context.Background()

	testCheck := func(*terraform.State) error {
		client := testAccProvider.Meta().(*ProviderConfig).DockerClient

		createExecOpts := types.ExecConfig{
			Cmd: []string{"dd", "if=/dev/zero_test", "of=/tmp/test.txt", "count=10", "bs=1"},
		}

		exec, err := client.ContainerExecCreate(ctx, c.ID, createExecOpts)
		if err != nil {
			return fmt.Errorf("Unable to create a exec instance on container: %s", err)
		}

		startExecOpts := types.ExecStartCheck{}
		if err := client.ContainerExecStart(ctx, exec.ID, startExecOpts); err != nil {
			return fmt.Errorf("Unable to run exec a instance on container: %s", err)
		}

		srcPath := "/tmp/test.txt"
		out, _, err := client.CopyFromContainer(ctx, c.ID, srcPath)
		if err != nil {
			return fmt.Errorf("Unable to download a file from container: %s", err)
		}

		tr := tar.NewReader(out)
		if _, err := tr.Next(); err != nil {
			return fmt.Errorf("Unable to read content of tar archive: %s", err)
		}

		fbuf := new(bytes.Buffer)
		if _, err := fbuf.ReadFrom(tr); err != nil {
			return err
		}
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerDeviceConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerInternalPortConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerMultipleInternalPortConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerPortConfig"),
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerMultiplePortConfig"),
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
	ctx := context.Background()

	testCheck := func(*terraform.State) error {
		if !c.HostConfig.AutoRemove {
			return fmt.Errorf("Container doesn't have a correct autoremove flag")
		}

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccContainerWaitConditionRemoved(ctx, "docker_container.foo", &c),
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerRmConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerReadOnlyConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerHealthcheckConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerNoStartConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerAttachConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerLogsConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerExitCodeConfig"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerNetworksIPv4AddressConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttrSet("docker_container.foo", "network_data.0.mac_address"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerNetworksIPv6AddressConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
					resource.TestCheckResourceAttr("docker_container.foo", "network_data.0.global_ipv6_address", "fd00:0:0:0::123"),
					resource.TestCheckResourceAttr("docker_container.foo", "network_data.0.global_ipv6_prefix_length", "64"),
					resource.TestCheckResourceAttr("docker_container.foo", "network_data.0.ipv6_gateway", "fd00:0:0:0::f"),
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_container", "testAccDockerContainerNetworksDualStackAddressConfig"),
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
					resource.TestCheckResourceAttr("docker_container.foo", "name", "tf-test"),
				),
			},
		},
	})
}

// /////////
// HELPERS
// /////////
func testAccContainerRunning(resourceName string, container *types.ContainerJSON) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource with name '%s' not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		containers, err := client.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				inspected, err := client.ContainerInspect(ctx, c.ID)
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
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		containers, err := client.ContainerList(ctx, types.ContainerListOptions{
			All: true,
		})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				inspected, err := client.ContainerInspect(ctx, c.ID)
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
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		statusC, errC := client.ContainerWait(ctx, rs.Primary.ID, container.WaitConditionNotRunning)

		select {
		case err := <-errC:
			if err != nil {
				return fmt.Errorf("Container is still running")
			}
		case <-statusC:
		}

		return nil
	}
}

func testAccContainerWaitConditionRemoved(ctx context.Context, n string, ct *types.ContainerJSON) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		statusC, errC := client.ContainerWait(ctx, rs.Primary.ID, container.WaitConditionRemoved)

		select {
		case err := <-errC:
			if err != nil {
				if !containsIgnorableErrorMessage(err.Error(), "No such container", "is already in progress") {
					return fmt.Errorf("Container has not been removed: '%s'", err.Error())
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
