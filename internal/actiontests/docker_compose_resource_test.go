package actiontests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDockerComposeResource_basicUpdate(t *testing.T) {
	preCheckDocker(t)

	projectName := fmt.Sprintf("tfacc-docker-compose-%d", time.Now().UnixNano())
	fixturesDir := composeResourceFixturesDir(t)
	var web containertypes.InspectResponse
	var worker containertypes.InspectResponse

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             testCheckComposeProjectRemoved(projectName),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadComposeResourceTestConfiguration(t, "testAccDockerComposeConfig"), projectName, fixturesDir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_compose.test", "project_name", projectName),
					resource.TestCheckResourceAttr("docker_compose.test", "remove_orphans", "true"),
					testCheckComposeContainerRunning(projectName, "web", &web),
					testCheckComposeContainerRunning(projectName, "worker", &worker),
				),
			},
			{
				Config: fmt.Sprintf(loadComposeResourceTestConfiguration(t, "testAccDockerComposeUpdatedConfig"), projectName, fixturesDir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_compose.test", "project_name", projectName),
					testCheckComposeContainerRunning(projectName, "web", &web),
					testCheckComposeContainerAbsent(projectName, "worker"),
				),
			},
		},
	})
}

func TestDockerComposeResource_profilesAndEnvFiles(t *testing.T) {
	preCheckDocker(t)

	projectName := fmt.Sprintf("tfacc-docker-compose-profiles-%d", time.Now().UnixNano())
	fixturesDir := composeResourceFixturesDir(t)
	var app containertypes.InspectResponse
	var optional containertypes.InspectResponse

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             testCheckComposeProjectRemoved(projectName),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadComposeResourceTestConfiguration(t, "testAccDockerComposeProfilesConfig"), projectName, fixturesDir, fixturesDir),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_compose.test", "project_name", projectName),
					resource.TestCheckResourceAttr("docker_compose.test", "profiles.#", "1"),
					resource.TestCheckResourceAttr("docker_compose.test", "env_files.#", "1"),
					testCheckComposeContainerRunning(projectName, "app", &app),
					testCheckComposeContainerRunning(projectName, "optional", &optional),
					testCheckComposeContainerHasEnv(projectName, "app", "COMPOSE_MESSAGE=from-env-file"),
				),
			},
		},
	})
}

func loadComposeResourceTestConfiguration(t *testing.T, testName string) string {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	configurationPath := filepath.Join(workingDir, "..", "..", "testdata", "resources", "docker_compose", testName+".tf")
	configurationContent, err := os.ReadFile(configurationPath)
	if err != nil {
		t.Fatalf("failed to read test configuration at %q: %v", configurationPath, err)
	}

	return string(configurationContent)
}

func composeResourceFixturesDir(t *testing.T) string {
	t.Helper()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	return filepath.Join(workingDir, "..", "..", "testdata", "resources", "docker_compose")
}

func testCheckComposeContainerRunning(projectName string, serviceName string, runningContainer *containertypes.InspectResponse) resource.TestCheckFunc {
	return func(*terraform.State) error {
		inspected, err := inspectComposeContainer(projectName, serviceName)
		if err != nil {
			return err
		}

		if !inspected.State.Running {
			return fmt.Errorf("compose container %q for project %q is not running", serviceName, projectName)
		}

		*runningContainer = inspected
		return nil
	}
}

func testCheckComposeContainerAbsent(projectName string, serviceName string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := newDockerClient()
		if err != nil {
			return err
		}
		defer client.Close() // nolint:errcheck

		containers, err := client.ContainerList(context.Background(), containertypes.ListOptions{
			All:     true,
			Filters: composeContainerFilters(projectName, serviceName),
		})
		if err != nil {
			return err
		}

		if len(containers) != 0 {
			return fmt.Errorf("expected compose service %q for project %q to be removed, found %d container(s)", serviceName, projectName, len(containers))
		}

		return nil
	}
}

func testCheckComposeContainerHasEnv(projectName string, serviceName string, wantedEnv string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		inspected, err := inspectComposeContainer(projectName, serviceName)
		if err != nil {
			return err
		}

		for _, env := range inspected.Config.Env {
			if env == wantedEnv {
				return nil
			}
		}

		return fmt.Errorf("compose container %q for project %q does not contain env %q: %#v", serviceName, projectName, wantedEnv, inspected.Config.Env)
	}
}

func testCheckComposeProjectRemoved(projectName string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := newDockerClient()
		if err != nil {
			return err
		}
		defer client.Close() // nolint:errcheck

		containers, err := client.ContainerList(context.Background(), containertypes.ListOptions{
			All: true,
			Filters: filters.NewArgs(
				filters.Arg("label", "com.docker.compose.project="+projectName),
			),
		})
		if err != nil {
			return err
		}

		if len(containers) != 0 {
			serviceNames := make([]string, 0, len(containers))
			for _, container := range containers {
				serviceNames = append(serviceNames, container.Labels["com.docker.compose.service"])
			}
			return fmt.Errorf("compose project %q still has containers: %s", projectName, strings.Join(serviceNames, ", "))
		}

		return nil
	}
}

func inspectComposeContainer(projectName string, serviceName string) (containertypes.InspectResponse, error) {
	client, err := newDockerClient()
	if err != nil {
		return containertypes.InspectResponse{}, err
	}
	defer client.Close() // nolint:errcheck

	containers, err := client.ContainerList(context.Background(), containertypes.ListOptions{
		All:     true,
		Filters: composeContainerFilters(projectName, serviceName),
	})
	if err != nil {
		return containertypes.InspectResponse{}, err
	}

	if len(containers) == 0 {
		return containertypes.InspectResponse{}, fmt.Errorf("compose service %q for project %q not found", serviceName, projectName)
	}

	inspected, err := client.ContainerInspect(context.Background(), containers[0].ID)
	if err != nil {
		return containertypes.InspectResponse{}, fmt.Errorf("container could not be inspected: %s", err)
	}

	return inspected, nil
}

func composeContainerFilters(projectName string, serviceName string) filters.Args {
	return filters.NewArgs(
		filters.Arg("label", "com.docker.compose.project="+projectName),
		filters.Arg("label", "com.docker.compose.service="+serviceName),
	)
}

func newDockerClient() (*dockerclient.Client, error) {
	return dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
}
