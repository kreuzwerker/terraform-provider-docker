package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"context"

	"github.com/docker/docker/api/types"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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

func TestAccDockerContainer_basic(t *testing.T) {
	var c types.ContainerJSON
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
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
			resource.TestStep{
				Config: testAccDockerContainerVolumeConfig,
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

		if c.HostConfig.CPUShares != 32 {
			return fmt.Errorf("Container has wrong cpu shares setting: %d", c.HostConfig.CPUShares)
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

		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerCustomizedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
			},
		},
	})
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
			resource.TestStep{
				Config: testAccDockerContainerUploadConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerRunning("docker_container.foo", &c),
					testCheck,
				),
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
			resource.TestStep{
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
			resource.TestStep{
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
			resource.TestStep{
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
			resource.TestStep{
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
			resource.TestStep{
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

func TestAccDockerContainer_nostart(t *testing.T) {
	var c types.ContainerJSON

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccDockerContainerNoStartConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccContainerNotRunning("docker_container.foo", &c),
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
		containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			return err
		}

		for _, c := range containers {
			if c.ID == rs.Primary.ID {
				return fmt.Errorf("Container found: %s", rs.Primary.ID)
			}
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

const testAccDockerContainerConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "${docker_image.foo.latest}"
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
	memory_swap = 2048
	cpu_shares = 32

	capabilities {
		add= ["ALL"]
		drop = ["SYS_ADMIN"]
	}

	dns = ["8.8.8.8"]
	dns_opts = ["rotate"]
	dns_search = ["example.com"]
	labels {
		env = "prod"
		role = "test"
	}
	log_driver = "json-file"
	log_opts = {
		max-size = "10m"
		max-file = 20
	}
	network_mode = "bridge"

	networks = ["${docker_network.test_network.name}"]
	network_alias = ["tftest"]

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
}

resource "docker_network" "test_network" {
  name = "test"
}
`

const testAccDockerContainerUploadConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
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
		internal = "80"
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
	
	ports = [
		{
			internal = "80"
		},
		{
			internal = "81"
		}
	]
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
		internal = "80"
		external = "32787"
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

	ports = [
		{
			internal = "80"
			external = "32787"
		},
		{
			internal = "81"
			external = "32788"
		}
	] 
}
`
const testAccDockerContainerNoStartConfig = `
resource "docker_image" "foo" {
	name = "nginx:latest"
	keep_locally = true
}

resource "docker_container" "foo" {
	name = "tf-test"
	image = "nginx:latest"
	start = false
    must_run = false
}
`
