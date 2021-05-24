package provider

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var contentDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerImage_basic(t *testing.T) {
	// run a Docker container which refers the Docker image to test "force_remove" option
	containerName := "test-docker-image-force-remove"
	ctx := context.Background()
	if err := exec.Command("docker", "run", "--rm", "-d", "--name", containerName, "alpine:3.11.5", "tail", "-f", "/dev/null").Run(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := exec.Command("docker", "stop", containerName).Run(); err != nil {
			t.Logf("failed to stop the Docker container %s: %v", containerName, err)
		}
	}()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo", "latest", contentDigestRegexp),
				),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccForceRemoveDockerImage"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.test", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_private(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAddDockerPrivateImageConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobar", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_destroy(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(s *terraform.State) error {
			for _, rs := range s.RootModule().Resources {
				if rs.Type != "docker_image" {
					continue
				}

				client := testAccProvider.Meta().(*ProviderConfig).DockerClient
				_, _, err := client.ImageInspectWithRaw(ctx, rs.Primary.Attributes["name"])
				if err != nil {
					return err
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageKeepLocallyConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobarzoo", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_data(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         providerFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageFromDataConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobarbaz", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_data_pull_trigger(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         providerFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageFromDataConfigWithPullTrigger"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobarbazoo", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_data_private(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	ctx := context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         providerFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageFromDataPrivateConfig"), registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo_private", "latest", contentDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerImage_data_private_config_file(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	wd, _ := os.Getwd()
	dockerConfig := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "dockerconfig.json"), "\\", "\\\\")
	ctx := context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         providerFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageFromDataPrivateConfigFile"), registry, dockerConfig, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo_private", "latest", contentDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerImage_data_private_config_file_content(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	wd, _ := os.Getwd()
	dockerConfig := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "dockerconfig.json"), "\\", "\\\\")
	ctx := context.Background()

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         providerFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageFromDataPrivateConfigFileContent"), registry, dockerConfig, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo_private", "latest", contentDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerImage_sha265(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAddDockerImageWithSHA256RepoDigest"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobar", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func testAccDockerImageDestroy(ctx context.Context, s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "docker_image" {
			continue
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		_, _, err := client.ImageInspectWithRaw(ctx, rs.Primary.Attributes["name"])
		if err == nil {
			return fmt.Errorf("Image still exists")
		}
	}
	return nil
}

func TestAccDockerImage_tag_sha265(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageWithTagAndSHA256RepoDigest"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.nginx", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_build(t *testing.T) {
	ctx := context.Background()
	wd, _ := os.Getwd()
	dfPath := filepath.Join(wd, "Dockerfile")
	if err := ioutil.WriteFile(dfPath, []byte(testDockerFileExample), 0o644); err != nil {
		t.Fatalf("failed to create a Dockerfile %s for test: %+v", dfPath, err)
	}
	defer os.Remove(dfPath)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testCreateDockerImage"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.test", "name", contentDigestRegexp),
				),
			},
		},
	})
}

const testDockerFileExample = `
FROM python:3-stretch

WORKDIR /app

ARG test_arg

RUN echo ${test_arg} > test_arg.txt

RUN apt-get update -qq
`
