package docker

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var contentDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

const testForceRemoveDockerImageName = "alpine:3.11.5"

func TestAccDockerImage_basic(t *testing.T) {
	// run a Docker container which refers the Docker image to test "force_remove" option
	containerName := "test-docker-image-force-remove"
	if err := exec.Command("docker", "run", "--rm", "-d", "--name", containerName, testForceRemoveDockerImageName, "tail", "-f", "/dev/null").Run(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := exec.Command("docker", "stop", containerName).Run(); err != nil {
			t.Logf("failed to stop the Docker container %s: %v", containerName, err)
		}
	}()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo", "latest", contentDigestRegexp),
				),
			},
			{
				Config: testAccForceRemoveDockerImage,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.test", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAddDockerPrivateImageConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobar", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_destroy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(s *terraform.State) error {
			for _, rs := range s.RootModule().Resources {
				if rs.Type != "docker_image" {
					continue
				}

				client := testAccProvider.Meta().(*ProviderConfig).DockerClient
				_, _, err := client.ImageInspectWithRaw(context.Background(), rs.Primary.Attributes["name"])
				if err != nil {
					return err
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccDockerImageKeepLocallyConfig,
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
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerImageFromDataConfig,
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
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: testAccDockerImageFromDataConfigWithPullTrigger,
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

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDockerImageFromDataPrivateConfig, registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo_private", "latest", contentDigestRegexp),
				),
			},
		},
		CheckDestroy: checkAndRemoveImages,
	})
}

func TestAccDockerImage_data_private_config_file(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	wd, _ := os.Getwd()
	dockerConfig := wd + "/../scripts/testing/dockerconfig.json"

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDockerImageFromDataPrivateConfigFile, registry, dockerConfig, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo_private", "latest", contentDigestRegexp),
				),
			},
		},
		CheckDestroy: checkAndRemoveImages,
	})
}

func TestAccDockerImage_data_private_config_file_content(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	wd, _ := os.Getwd()
	dockerConfig := wd + "/../scripts/testing/dockerconfig.json"

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDockerImageFromDataPrivateConfigFileContent, registry, dockerConfig, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foo_private", "latest", contentDigestRegexp),
				),
			},
		},
		CheckDestroy: checkAndRemoveImages,
	})
}

func TestAccDockerImage_sha265(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAddDockerImageWithSHA256RepoDigest,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.foobar", "latest", contentDigestRegexp),
				),
			},
		},
	})
}

func testAccDockerImageDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "docker_image" {
			continue
		}

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		_, _, err := client.ImageInspectWithRaw(context.Background(), rs.Primary.Attributes["name"])
		if err == nil {
			return fmt.Errorf("Image still exists")
		}
	}
	return nil
}

func TestAccDockerImage_build(t *testing.T) {
	wd, _ := os.Getwd()
	dfPath := path.Join(wd, "Dockerfile")
	if err := ioutil.WriteFile(dfPath, []byte(testDockerFileExample), 0o644); err != nil {
		t.Fatalf("failed to create a Dockerfile %s for test: %+v", dfPath, err)
	}
	defer os.Remove(dfPath)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccDockerImageDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCreateDockerImage,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.test", "name", contentDigestRegexp),
				),
			},
		},
	})
}

const testAccDockerImageConfig = `
resource "docker_image" "foo" {
	name = "alpine:3.1"
}
`

const testAccForceRemoveDockerImage = `
resource "docker_image" "test" {
	name         = "` + testForceRemoveDockerImageName + `"
	force_remove = true
}
`

const testAddDockerPrivateImageConfig = `
resource "docker_image" "foobar" {
	name = "gcr.io:443/google_containers/pause:0.8.0"
}
`

const testAccDockerImageKeepLocallyConfig = `
resource "docker_image" "foobarzoo" {
	name = "crux:3.1"
	keep_locally = true
}
`

const testAccDockerImageFromDataConfig = `
data "docker_registry_image" "foobarbaz" {
	name = "alpine:3.1"
}
resource "docker_image" "foobarbaz" {
	name = "${data.docker_registry_image.foobarbaz.name}"
	pull_triggers = ["${data.docker_registry_image.foobarbaz.sha256_digest}"]
}
`

const testAccDockerImageFromDataConfigWithPullTrigger = `
data "docker_registry_image" "foobarbazoo" {
	name = "alpine:3.1"
}
resource "docker_image" "foobarbazoo" {
	name = "${data.docker_registry_image.foobarbazoo.name}"
	pull_trigger = "${data.docker_registry_image.foobarbazoo.sha256_digest}"
}
`

const testAccDockerImageFromDataPrivateConfig = `
provider "docker" {
	alias = "private"
	registry_auth {
		address = "%s"
	}
}
data "docker_registry_image" "foo_private" {
	provider = "docker.private"
	name = "%s"
}
resource "docker_image" "foo_private" {
	provider = "docker.private"
	name = "${data.docker_registry_image.foo_private.name}"
	keep_locally = true
	pull_triggers = ["${data.docker_registry_image.foo_private.sha256_digest}"]
}
`

const testAccDockerImageFromDataPrivateConfigFile = `
provider "docker" {
	alias = "private"
	registry_auth {
		address = "%s"
		config_file = "%s"
	}
}
resource "docker_image" "foo_private" {
	provider = "docker.private"
	name = "%s"
}
`

const testAccDockerImageFromDataPrivateConfigFileContent = `
provider "docker" {
	alias = "private"
	registry_auth {
		address = "%s"
		config_file_content = "${file("%s")}"
	}
}
resource "docker_image" "foo_private" {
	provider = "docker.private"
	name = "%s"
}
`

const testAddDockerImageWithSHA256RepoDigest = `
resource "docker_image" "foobar" {
	name = "stocard/gotthard@sha256:ed752380c07940c651b46c97ca2101034b3be112f4d86198900aa6141f37fe7b"
}
`

const testCreateDockerImage = `
resource "docker_image" "test" {
	name = "ubuntu:11"
	build {
	  path = "."
	  dockerfile = "Dockerfile"
	  force_remove = true
	  build_arg = {
		test_arg = "kenobi"
	  }
	  label = {
		test_label1 = "han"
		test_label2 = "solo"
	  }
	}
  }  
`

const testDockerFileExample = `
FROM python:3-stretch

WORKDIR /app

ARG test_arg

RUN echo ${test_arg} > test_arg.txt

RUN apt-get update -qq
`
