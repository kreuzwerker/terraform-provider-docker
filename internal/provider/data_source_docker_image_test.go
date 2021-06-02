package provider

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var imageDigestRegexp = regexp.MustCompile(`^sha256:[A-Fa-f0-9]+$`)
var imageNameWithTagAndDigestRegexp = regexp.MustCompile(`^.*:.*@sha256:[A-Fa-f0-9]+$`)

func TestAccDockerImageDataSource_withSpecificTag(t *testing.T) {
	ctx := context.Background()
	imageName := "nginx:1.17.6"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			pullImageForTest(t, imageName)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_image", "testAccDockerImageDataSourceWithSpecificTag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_image.foo", "name", imageName),
					resource.TestCheckResourceAttr("data.docker_image.foo", "latest", "sha256:f7bb5701a33c0e572ed06ca554edca1bee96cbbc1f76f3b01c985de7e19d0657"),
					resource.TestCheckResourceAttr("data.docker_image.foo", "sha256_digest", "sha256:f7bb5701a33c0e572ed06ca554edca1bee96cbbc1f76f3b01c985de7e19d0657"),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return removeImageForTest(ctx, state, imageName)
		},
	})
}

func TestAccDockerImageDataSource_withDefaultTag(t *testing.T) {
	ctx := context.Background()
	imageName := "nginx"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			pullImageForTest(t, imageName)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_image", "testAccDockerImageDataSourceWithDefaultTag"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_image.foo", "name", imageName),
					resource.TestMatchResourceAttr("data.docker_image.foo", "latest", imageDigestRegexp),
					resource.TestMatchResourceAttr("data.docker_image.foo", "sha256_digest", imageDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return removeImageForTest(ctx, state, imageName)
		},
	})
}

func TestAccDockerImageDataSource_withSha256Digest(t *testing.T) {
	ctx := context.Background()
	imageName := "nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			pullImageForTest(t, imageName)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_image", "testAccDockerImageDataSourceWithSha256Digest"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_image.foo", "name", imageName),
					resource.TestMatchResourceAttr("data.docker_image.foo", "latest", imageDigestRegexp),
					resource.TestMatchResourceAttr("data.docker_image.foo", "sha256_digest", imageDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return removeImageForTest(ctx, state, imageName)
		},
	})
}
func TestAccDockerImageDataSource_withTagAndSha256Digest(t *testing.T) {
	ctx := context.Background()
	imageName := "nginx:1.19.1@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			pullImageForTest(t, imageName)
		},
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_image", "testAccDockerImageDataSourceWithTagAndSha256Digest"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.docker_image.foo", "name", imageName),
					resource.TestMatchResourceAttr("data.docker_image.foo", "latest", imageDigestRegexp),
					resource.TestMatchResourceAttr("data.docker_image.foo", "sha256_digest", imageDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return removeImageForTest(ctx, state, imageName)
		},
	})
}

func TestAccDockerImageDataSource_withNonExistentImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				data "docker_image" "foo" {
					name = "image-does-not-exist"
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not find docker image.*`),
			},
			{
				Config: `
				data "docker_image" "foo" {
					name = "nginx:tag-does-not-exist"
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not find docker image.*`),
			},
			{
				Config: `
				data "docker_image" "foo" {
					name = "nginx@shaDoesNotExist"
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not find docker image.*`),
			},
			{
				Config: `
				data "docker_image" "foo" {
					name = "nginx:1.19.1@shaDoesNotExist"
				}
				`,
				ExpectError: regexp.MustCompile(`.*did not find docker image.*`),
			},
		},
	})
}

// Helpers
func pullImageForTest(t *testing.T, imageName string) {
	cmd := exec.Command("docker", "pull", imageName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to pull image '%s': %s", imageName, err)
	}
}

func removeImageForTest(ctx context.Context, s *terraform.State, imageName string) error {
	client := testAccProvider.Meta().(*ProviderConfig).DockerClient

	// for images with tag and digest like e.g.
	// 'nginx:1.19.1@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2'
	// no image is found. This is why we strip the tag to remain with
	// 'nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2'
	if imageNameWithTagAndDigestRegexp.MatchString(imageName) {
		tagStartIndex := strings.Index(imageName, ":")
		digestStartIndex := strings.Index(imageName, "@")
		imageName = imageName[:tagStartIndex] + imageName[digestStartIndex:]
	}

	filters := filters.NewArgs()
	filters.Add("reference", imageName)
	images, err := client.ImageList(ctx, types.ImageListOptions{
		Filters: filters,
	})
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return fmt.Errorf("did not find any image with name '%s' to delete", imageName)
	}

	for _, image := range images {
		_, err := client.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{
			Force: true,
		})
		if err != nil {
			return fmt.Errorf("failed to remove image with ID '%s'", image.ID)
		}
	}

	return nil
}
