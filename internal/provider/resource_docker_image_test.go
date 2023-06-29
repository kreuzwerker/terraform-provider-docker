package provider

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var contentDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerRegistryImageResource_mapping(t *testing.T) {
	assert := func(condition bool, msg string) {
		if !condition {
			t.Errorf("assertion failed: wrong build parameter %s", msg)
		}
	}

	dummyProvider := New("dev")()
	dummyResource := dummyProvider.ResourcesMap["docker_image"]
	dummyResource.CreateContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

		if value, ok := d.GetOk("build"); ok {
			for _, rawBuild := range value.(*schema.Set).List() {
				build := rawBuild.(map[string]interface{})
				// build := d.Get("build").([]interface{})[0].(map[string]interface{})
				options := createImageBuildOptions(build)

				assert(options.SuppressOutput == true, "SuppressOutput")
				assert(options.RemoteContext == "fooRemoteContext", "RemoteContext")
				assert(options.NoCache == true, "NoCache")
				assert(options.Remove == true, "Remove")
				assert(options.ForceRemove == true, "ForceRemove")
				assert(options.PullParent == true, "PullParent")
				assert(options.Isolation == container.Isolation("hyperv"), "Isolation")
				assert(options.CPUSetCPUs == "fooCpuSetCpus", "CPUSetCPUs")
				assert(options.CPUSetMems == "fooCpuSetMems", "CPUSetMems")
				assert(options.CPUShares == int64(4), "CPUShares")
				assert(options.CPUQuota == int64(5), "CPUQuota")
				assert(options.CPUPeriod == int64(6), "CPUPeriod")
				assert(options.Memory == int64(1), "Memory")
				assert(options.MemorySwap == int64(2), "MemorySwap")
				assert(options.CgroupParent == "fooCgroupParent", "CgroupParent")
				assert(options.NetworkMode == "fooNetworkMode", "NetworkMode")
				assert(options.ShmSize == int64(3), "ShmSize")
				assert(options.Dockerfile == "fooDockerfile", "Dockerfile")
				assert(len(options.Ulimits) == 1, "Ulimits")
				assert(reflect.DeepEqual(*options.Ulimits[0], units.Ulimit{
					Name: "foo",
					Hard: int64(1),
					Soft: int64(2),
				}), "Ulimits")
				assert(len(options.BuildArgs) == 1, "BuildArgs")
				// DevSkim: ignore DS137138
				assert(*options.BuildArgs["HTTP_PROXY"] == "http://10.20.30.2:1234", "BuildArgs")
				assert(len(options.AuthConfigs) == 1, "AuthConfigs")
				assert(reflect.DeepEqual(options.AuthConfigs["foo.host"], types.AuthConfig{
					Username:      "fooUserName",
					Password:      "fooPassword",
					Auth:          "fooAuth",
					Email:         "fooEmail",
					ServerAddress: "fooServerAddress",
					IdentityToken: "fooIdentityToken",
					RegistryToken: "fooRegistryToken",
				}), "AuthConfigs")
				assert(reflect.DeepEqual(options.Labels, map[string]string{"foo": "bar"}), "Labels")
				assert(options.Squash == true, "Squash")
				assert(reflect.DeepEqual(options.CacheFrom, []string{"fooCacheFrom", "barCacheFrom"}), "CacheFrom")
				assert(reflect.DeepEqual(options.SecurityOpt, []string{"fooSecurityOpt", "barSecurityOpt"}), "SecurityOpt")
				assert(reflect.DeepEqual(options.ExtraHosts, []string{"fooExtraHost", "barExtraHost"}), "ExtraHosts")
				assert(options.Target == "fooTarget", "Target")
				assert(options.SessionID == "fooSessionId", "SessionID")
				assert(options.Platform == "fooPlatform", "Platform")
				assert(options.Version == types.BuilderVersion("1"), "Version")
				assert(options.BuildID == "fooBuildId", "BuildID")
				// output
				d.SetId("foo")
			}
		}
		return nil
	}
	dummyResource.UpdateContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		return nil
	}
	dummyResource.DeleteContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		return nil
	}
	dummyResource.ReadContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		d.Set("id", "foo")
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"docker": func() (*schema.Provider, error) {
				return dummyProvider, nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testBuildDockerImageMappingConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_image.foo", "id"),
				),
			},
		},
	})
}

func TestAccDockerImage_basic(t *testing.T) {
	// run a Docker container which refers the Docker image to test "force_remove" option
	containerName := "test-docker-image-force-remove"
	ctx := context.Background()
	if err := exec.Command("docker", "run", "--rm", "-d", "--name", containerName, "alpine:3.16.0", "tail", "-f", "/dev/null").Run(); err != nil {
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
					resource.TestMatchResourceAttr("docker_image.foo", "repo_digest", imageRepoDigestRegexp),
				),
			},
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccForceRemoveDockerImage"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.test", "repo_digest", imageRepoDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_private(t *testing.T) {
	ctx := context.Background()
	var i types.ImageInspect

	testCheckImageInspect := func(*terraform.State) error {
		if len(i.RepoTags) != 1 ||
			i.RepoTags[0] != "gcr.io:443/google_containers/pause:0.8.0" {
			return fmt.Errorf("Image RepoTags is wrong: %v", i.RepoTags)
		}

		if len(i.RepoDigests) != 1 ||
			i.RepoDigests[0] != "gcr.io:443/google_containers/pause@sha256:bbeaef1d40778579b7b86543fe03e1ec041428a50d21f7a7b25630e357ec9247" {
			return fmt.Errorf("Image RepoDigests is wrong: %v", i.RepoDigests)
		}

		return nil
	}

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
					resource.TestMatchResourceAttr("docker_image.foobar", "repo_digest", imageRepoDigestRegexp),
					testAccImageCreated("docker_image.foobar", &i),
					testCheckImageInspect,
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
					resource.TestMatchResourceAttr("docker_image.foobarzoo", "repo_digest", imageRepoDigestRegexp),
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
					resource.TestMatchResourceAttr("docker_image.foobarbaz", "repo_digest", imageRepoDigestRegexp),
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
					resource.TestMatchResourceAttr("docker_image.foobarbazoo", "repo_digest", imageRepoDigestRegexp),
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
					resource.TestMatchResourceAttr("docker_image.foo_private", "repo_digest", imageRepoDigestRegexp),
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
					resource.TestMatchResourceAttr("docker_image.foo_private", "repo_digest", imageRepoDigestRegexp),
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
					resource.TestMatchResourceAttr("docker_image.foo_private", "repo_digest", imageRepoDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

// Changing the name attribute should also force a change of the dependent docker container
// This test fails, if we remove the ForceTrue: true from the name attribute
func TestAccDockerImage_name_attr_change(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         providerFactories,
		PreventPostDestroyRefresh: true,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageName"), "ubuntu:precise@sha256:18305429afa14ea462f810146ba44d4363ae76e4c8dfc38288cf73aa07485005"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.ubuntu", "repo_digest", imageRepoDigestRegexp),
				),
			},
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImageName"), "ubuntu:jammy@sha256:b6b83d3c331794420340093eb706a6f152d9c1fa51b262d9bf34594887c2c7ac"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.ubuntu", "repo_digest", imageRepoDigestRegexp),
				),
			},
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
					resource.TestMatchResourceAttr("docker_image.foobar", "repo_digest", imageRepoDigestRegexp),
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
					resource.TestMatchResourceAttr("docker_image.nginx", "repo_digest", imageRepoDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerImage_platform(t *testing.T) {
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testAccDockerImagePlatform"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docker_image.foo", "image_id", "sha256:8336f9f1d0946781f428a155536995f0d8a31209d65997e2a379a23e7a441b78"),
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

func TestAccDockerImage_buildTimeout(t *testing.T) {
	ctx := context.Background()
	wd, _ := os.Getwd()

	createTimeout := time.Duration(1 * time.Second)
	buildSleep := time.Duration(2 * time.Second)
	if buildSleep < createTimeout {
		t.Errorf("assertion failed: build sleep can't be shorter than the timeout")
	}

	dfPath := filepath.Join(wd, "Dockerfile")
	dfContent := []byte(fmt.Sprintf("FROM alpine\nRUN sleep %d", int(buildSleep.Seconds())))
	if err := ioutil.WriteFile(dfPath, dfContent, 0o644); err != nil {
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
				Config: fmt.Sprintf(
					loadTestConfiguration(t, RESOURCE, "docker_image", "testCreateDockerImageCreateTimeout"),
					createTimeout,
				),
				ExpectError: regexp.MustCompile("deadline exceeded"),
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

// Test for implementation of https://github.com/kreuzwerker/terraform-provider-docker/issues/401
func TestAccDockerImage_buildOutsideContext(t *testing.T) {
	ctx := context.Background()
	wd, _ := os.Getwd()
	dfPath := filepath.Join(wd, "..", "Dockerfile")
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_image", "testDockerImageDockerfileOutsideContext"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("docker_image.outside_context", "name", regexp.MustCompile(`\Aoutside-context:latest\z`)),
				),
			},
		},
	})
}

func TestAccDockerImageResource_build(t *testing.T) {
	name := "tftest-dockerregistryimage:1.0"
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testBuildDockerImageNoKeepConfig"), name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_image.foo", "image_id"),
				),
			},
		},
	})
}

// Test for https://github.com/kreuzwerker/terraform-provider-docker/issues/249
func TestAccDockerImageResource_whitelistDockerignore(t *testing.T) {
	name := "tftest-dockerregistryimage-whitelistdockerignore:1.0"
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_file_whitelist_dockerignore")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testDockerImageFilePermissions"), name, context, "Dockerfile"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_image.file_permissions", "image_id"),
				),
			},
		},
	})
}

// Tests for issue https://github.com/kreuzwerker/terraform-provider-docker/issues/293
// First we check if we can build the docker_registry_image resource at all
// TODO in a second step we want to check whether the file has the correct permissions
func TestAccDockerImageResource_correctFilePermissions(t *testing.T) {
	name := "tftest-dockerregistryimage-filepermissions:1.0"
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_file_permissions")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testDockerImageFilePermissions"), name, context, "Dockerfile"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_image.file_permissions", "image_id"),
				),
				// TODO another check which starts the the newly built docker image and checks the file permissions to see if they are correct
			},
		},
	})
}

func TestAccDockerImageResource_buildWithDockerignore(t *testing.T) {
	name := "tftest-dockerregistryimage-ignore:1.0"
	wd, _ := os.Getwd()
	ctx := context.Background()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context_dockerignore")), "\\", "\\\\")
	ignoredFile := context + "/to_be_ignored"
	expectedSha := ""

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testBuildDockerImageNoKeepJustCache"), "one", name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_image.one", "image_id"),
					resource.TestCheckResourceAttrWith("docker_image.one", "image_id", func(value string) error {
						expectedSha = value
						return nil
					}),
				),
			},
			{
				PreConfig: func() {
					// create a file that should be ignored
					f, err := os.OpenFile(ignoredFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						panic("failed to create test file")
					}
					f.Close()
				},
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_image", "testBuildDockerImageNoKeepJustCache"), "two", name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("docker_image.two", "image_id", func(value string) error {
						if value != expectedSha {
							return fmt.Errorf("Image sha256_digest changed, expected %#v, got %#v", expectedSha, value)
						}
						return nil
					}),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return testAccDockerImageDestroy(ctx, state)
		},
	})
}

func testAccImageCreated(resourceName string, image *types.ImageInspect) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource with name '%s' not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		name := rs.Primary.Attributes["name"]
		// TODO mavogel: it's because we set the ID in the format:
		// d.SetId(foundImage.ID + d.Get("name").(string))
		// so we need to strip away the name
		strippedID := strings.Replace(rs.Primary.ID, name, "", -1)

		client := testAccProvider.Meta().(*ProviderConfig).DockerClient
		inspectedImage, _, err := client.ImageInspectWithRaw(ctx, strippedID)
		if err != nil {
			return fmt.Errorf("Image with ID '%s': %w", strippedID, err)
		}

		// we set the value to the pointer to be able to use the value
		// outside of the function
		*image = inspectedImage
		return nil

	}
}

func TestParseImageOptions(t *testing.T) {
	t.Run("Should parse image name with registry", func(t *testing.T) {
		expected := internalPullImageOptions{Registry: "registry.com", Repository: "image", Tag: "tag"}
		result := parseImageOptions("registry.com/image:tag")
		if !reflect.DeepEqual(expected, result) {
			t.Fatalf("Result %#v did not match expectation %#v", result, expected)
		}
	})
	t.Run("Should parse image name with registryPort", func(t *testing.T) {
		expected := internalPullImageOptions{Registry: "registry.com:8080", Repository: "image", Tag: "tag"}
		result := parseImageOptions("registry.com:8080/image:tag")
		if !reflect.DeepEqual(expected, result) {
			t.Fatalf("Result %#v did not match expectation %#v", result, expected)
		}
	})
	t.Run("Should parse image name with registry and proper repository", func(t *testing.T) {
		expected := internalPullImageOptions{Registry: "registry.com", Repository: "repo/image", Tag: "tag"}
		result := parseImageOptions("registry.com/repo/image:tag")
		if !reflect.DeepEqual(expected, result) {
			t.Fatalf("Result %#v did not match expectation %#v", result, expected)
		}
	})
	t.Run("Should parse image with no tag", func(t *testing.T) {
		expected := internalPullImageOptions{Registry: "registry.com", Repository: "repo/image", Tag: "latest"}
		result := parseImageOptions("registry.com/repo/image")
		if !reflect.DeepEqual(expected, result) {
			t.Fatalf("Result %#v did not match expectation %#v", result, expected)
		}
	})
	t.Run("Should parse image name without registry and default to docker registry", func(t *testing.T) {
		expected := internalPullImageOptions{Registry: "registry-1.docker.io", Repository: "library/image", Tag: "tag"}
		result := parseImageOptions("image:tag")
		if !reflect.DeepEqual(expected, result) {
			t.Fatalf("Result %#v did not match expectation %#v", result, expected)
		}
	})
}
