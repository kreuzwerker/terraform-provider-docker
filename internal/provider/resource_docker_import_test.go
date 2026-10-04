package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Import must adopt daemon resources, not recreate them to populate state.
func TestAccDockerContainer_importExisting(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC is required for Docker acceptance tests")
	}
	ctx := context.Background()
	// This container/image/network journey needs no Swarm resources.
	if err := testAccProvider.Configure(ctx, terraform.NewResourceConfigRaw(nil)); err != nil {
		t.Fatal(err)
	}
	client, err := testAccProvider.Meta().(*ProviderConfig).MakeClient(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	docker := func(args ...string) string {
		t.Helper()
		commandCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		output, err := exec.CommandContext(commandCtx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker %v: %v: %s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	name := fmt.Sprintf("tf-test-import-%d", time.Now().UnixNano())
	baseImage := "nginx:latest"
	imageRef := name + ":import"
	_, imageErr := client.ImageInspect(ctx, baseImage)
	if imageErr != nil && !errdefs.IsNotFound(imageErr) {
		t.Fatal(imageErr)
	}
	// Register cleanup before creating any fixture. Remove the base image
	// only when it was absent before this disposable-daemon test.
	var networkID, containerID string
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if containerID != "" {
			if err := client.ContainerRemove(cleanupCtx, containerID, container.RemoveOptions{Force: true}); err != nil && !errdefs.IsNotFound(err) {
				t.Errorf("remove owned import container: %v", err)
			}
		}
		if networkID != "" {
			if err := client.NetworkRemove(cleanupCtx, networkID); err != nil && !errdefs.IsNotFound(err) {
				t.Errorf("remove owned import network: %v", err)
			}
		}
		if _, err := client.ImageRemove(cleanupCtx, imageRef, image.RemoveOptions{}); err != nil && !errdefs.IsNotFound(err) {
			t.Errorf("remove owned import image reference: %v", err)
		}
		if errdefs.IsNotFound(imageErr) {
			if _, err := client.ImageRemove(cleanupCtx, baseImage, image.RemoveOptions{}); err != nil && !errdefs.IsNotFound(err) {
				t.Errorf("remove owned import image: %v", err)
			}
		}
	})
	if errdefs.IsNotFound(imageErr) {
		docker("pull", baseImage)
	}
	docker("tag", baseImage, imageRef)
	inspectedImage, err := client.ImageInspect(ctx, imageRef)
	if err != nil {
		t.Fatal(err)
	}
	networkID = docker("network", "create", "--label", "com.docker.compose.project="+name,
		"--label", "com.docker.compose.network=private", name)
	containerID = docker("run", "-d", "--name", name, "--network", networkID,
		"--network-alias", "application", "--network-alias", name,
		"--label", "com.docker.compose.project="+name, "--label", "import.empty=",
		"--env", "IMPORT_VALUE=explicit", "--publish", "127.0.0.1::80",
		"--log-driver", "json-file", "--log-opt", "max-size=10m", imageRef)
	initial, err := client.ContainerInspect(ctx, containerID)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := initial.NetworkSettings.Networks[name]
	if endpoint == nil {
		t.Fatal("pre-existing container did not join its owned network")
	}
	bindings := initial.NetworkSettings.Ports["80/tcp"]
	if len(bindings) != 1 || bindings[0].HostIP != "127.0.0.1" {
		t.Fatal("pre-existing container has no unique loopback port")
	}
	checkUnchanged := func(state *terraform.State) error {
		current, err := client.ContainerInspect(ctx, containerID)
		if err != nil {
			return err
		}
		joined := current.NetworkSettings.Networks[name]
		if current.ID != initial.ID || current.Image != initial.Image || !current.State.Running ||
			joined == nil || joined.NetworkID != networkID || joined.MacAddress != endpoint.MacAddress ||
			!reflectStringSetEqual(joined.Aliases, endpoint.Aliases) ||
			!reflectStringSetEqual(current.Config.Env, initial.Config.Env) ||
			!reflect.DeepEqual(current.Config.Labels, initial.Config.Labels) ||
			len(current.NetworkSettings.Ports["80/tcp"]) != 1 || current.NetworkSettings.Ports["80/tcp"][0] != bindings[0] {
			return fmt.Errorf("import or reapply changed the pre-existing container identity, endpoint, labels, environment or loopback binding")
		}
		currentImage, err := client.ImageInspect(ctx, imageRef)
		if err != nil || currentImage.ID != inspectedImage.ID {
			return fmt.Errorf("import or reapply changed the pre-existing image: %v", err)
		}
		currentNetwork, err := client.NetworkInspect(ctx, networkID, network.InspectOptions{})
		if err != nil || currentNetwork.ID != networkID || currentNetwork.Labels["com.docker.compose.project"] != name || currentNetwork.Labels["com.docker.compose.network"] != "private" {
			return fmt.Errorf("import or reapply changed the pre-existing network: %v", err)
		}
		if state != nil {
			if err := resource.TestCheckResourceAttr("docker_container.adopted", "id", containerID)(state); err != nil {
				return err
			}
			if err := resource.TestCheckResourceAttr("docker_image.adopted", "id", inspectedImage.ID+imageRef)(state); err != nil {
				return err
			}
			if err := resource.TestCheckResourceAttr("docker_network.adopted", "id", networkID)(state); err != nil {
				return err
			}
		}
		return nil
	}
	configuration := fmt.Sprintf(`
resource "docker_image" "adopted" {
  name = %q
  keep_locally = true
}
resource "docker_network" "adopted" {
  name = %q
  labels {
    label = "com.docker.compose.project"
    value = %q
  }
  labels {
    label = "com.docker.compose.network"
    value = "private"
  }
}
resource "docker_container" "adopted" {
  name = %q
  image = docker_image.adopted.name
  network_mode = %q
  stop_signal = %q
  destroy_grace_seconds = 10
  env = ["IMPORT_VALUE=explicit"]
  log_driver = "json-file"
  log_opts = { "max-size" = "10m" }
  labels {
    label = "com.docker.compose.project"
    value = %q
  }
  labels {
    label = "import.empty"
    value = ""
  }
  networks_advanced {
    name = docker_network.adopted.id
    aliases = [%q, "application"]
    mac_address = %q
  }
  ports {
    internal = 80
    external = %s
    ip = "127.0.0.1"
  }
}
`, imageRef, name, name, name, string(initial.HostConfig.NetworkMode), initial.Config.StopSignal, name,
		name, endpoint.MacAddress, bindings[0].HostPort)
	checkImported := func(states []*terraform.InstanceState) error {
		for _, expected := range []string{inspectedImage.ID + imageRef, networkID, containerID} {
			found := false
			for _, state := range states {
				if state.ID == expected {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("CLI import did not retain daemon resource identity %s", expected)
			}
		}
		return checkUnchanged(nil)
	}
	resource.Test(t, resource.TestCase{
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{Config: configuration, ResourceName: "docker_image.adopted", ImportState: true,
				ImportStateId: inspectedImage.ID, ExpectError: regexp.MustCompile("explicit tagged or digest image reference")},
			{Config: configuration, ResourceName: "docker_image.adopted", ImportState: true,
				ImportStateId: name + ":missing", ExpectError: regexp.MustCompile("no local Docker image matches")},
			{Config: configuration, ResourceName: "docker_image.adopted", ImportState: true,
				ImportStateId: imageRef, ImportStatePersist: true, PreConfig: func() {
					if err := checkUnchanged(nil); err != nil {
						t.Fatal(err)
					}
				}},
			{Config: configuration, ResourceName: "docker_network.adopted", ImportState: true,
				ImportStateId: networkID, ImportStatePersist: true},
			{Config: configuration, ResourceName: "docker_container.adopted", ImportState: true,
				ImportStateId: containerID, ImportStatePersist: true, ImportStateCheck: checkImported},
			{Config: configuration, PlanOnly: true},
			{Config: configuration, Check: checkUnchanged},
		},
		CheckDestroy: func(_ *terraform.State) error {
			if _, err := client.ContainerInspect(ctx, containerID); !errdefs.IsNotFound(err) {
				return fmt.Errorf("SDK destroy did not remove adopted container: %v", err)
			}
			if _, err := client.NetworkInspect(ctx, networkID, network.InspectOptions{}); !errdefs.IsNotFound(err) {
				return fmt.Errorf("SDK destroy did not remove adopted network: %v", err)
			}
			if currentImage, err := client.ImageInspect(ctx, imageRef); err != nil || currentImage.ID != inspectedImage.ID {
				return fmt.Errorf("SDK destroy did not preserve the adopted image: %v", err)
			}
			return nil
		},
	})
}

func reflectStringSetEqual(left, right []string) bool {
	// Ordering is irrelevant for these daemon sets.
	counts := make(map[string]int)
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
