package provider

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDockerRegistryImageResource_mapping(t *testing.T) {
	assert := func(condition bool, msg string) {
		if !condition {
			t.Errorf("assertion failed: wrong build parameter %s", msg)
		}
	}

	dummyProvider := New("dev")()
	dummyResource := dummyProvider.ResourcesMap["docker_registry_image"]
	dummyResource.CreateContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		build := d.Get("build").([]interface{})[0].(map[string]interface{})
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
		d.Set("sha256_digest", "bar")
		return nil
	}
	dummyResource.UpdateContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		return nil
	}
	dummyResource.DeleteContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		return nil
	}
	dummyResource.ReadContext = func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		d.Set("sha256_digest", "bar")
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
				Config: loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageMappingConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
	})
}

func TestAccDockerRegistryImageResource_build(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageNoKeepConfig"), pushOptions.Registry, pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
		CheckDestroy: testDockerRegistryImageNotInRegistry(pushOptions),
	})
}

func TestAccDockerRegistryImageResource_buildAndKeep(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll(filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context"), "\\", "\\\\")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageKeepConfig"), pushOptions.Registry, pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
		// as the providerConfig obtained from testAccProvider.Meta().(*ProviderConfig)
		// is empty after the test the credetials are passed here manually
		CheckDestroy: testDockerRegistryImageInRegistry("testuser", "testpwd", pushOptions, true),
	})
}

func TestAccDockerRegistryImageResource_buildWithDockerignore(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage-ignore:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_context_dockerignore")), "\\", "\\\\")
	ignoredFile := context + "/to_be_ignored"
	expectedSha := ""

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageNoKeepJustCache"), pushOptions.Registry, "one", pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.one", "sha256_digest"),
					resource.TestCheckResourceAttrWith("docker_registry_image.one", "sha256_digest", func(value string) error {
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
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testBuildDockerRegistryImageNoKeepJustCache"), pushOptions.Registry, "two", pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("docker_registry_image.two", "sha256_digest", func(value string) error {
						if value != expectedSha {
							return fmt.Errorf("Image sha256_digest changed, expected %#v, got %#v", expectedSha, value)
						}
						return nil
					}),
				),
			},
		},
		CheckDestroy: resource.ComposeTestCheckFunc(
			testDockerRegistryImageNotInRegistry(pushOptions),
			func(*terraform.State) error {
				err := os.Remove(ignoredFile)
				if err != nil {
					return fmt.Errorf("failed to remove ignored file: %w", err)
				}
				return nil
			},
		),
	})
}

// Tests for issue https://github.com/kreuzwerker/terraform-provider-docker/issues/293
// First we check if we can build the docker_registry_image resource at all
// TODO in a second step we want to check whether the file has the correct permissions
func TestAccDockerRegistryImageResource_correctFilePermissions(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage-filepermissions:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_file_permissions")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testDockerRegistryImageFilePermissions"), pushOptions.Registry, pushOptions.Name, context, "Dockerfile"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.file_permissions", "sha256_digest"),
				),
				// TODO another check which starts the the newly built docker image and checks the file permissions to see if they are correct
			},
		},
	})
}

// Test for https://github.com/kreuzwerker/terraform-provider-docker/issues/249
func TestAccDockerRegistryImageResource_whitelistDockerignore(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage-whitelistdockerignore:1.0")
	wd, _ := os.Getwd()
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_file_whitelist_dockerignore")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testDockerRegistryImageFilePermissions"), pushOptions.Registry, pushOptions.Name, context, "Dockerfile"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.file_permissions", "sha256_digest"),
				),
			},
		},
	})
}

// Test for https://github.com/kreuzwerker/terraform-provider-docker/issues/249
func TestAccDockerRegistryImageResource_DockerfileOutsideContext(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage-dockerfileoutsidecontext:1.0")
	wd, _ := os.Getwd()
	dfPath := filepath.Join(wd, "..", "Dockerfile")
	if err := ioutil.WriteFile(dfPath, []byte(testDockerFileExample), 0o644); err != nil {
		t.Fatalf("failed to create a Dockerfile %s for test: %+v", dfPath, err)
	}
	defer os.Remove(dfPath)
	context := strings.ReplaceAll((filepath.Join(wd, "..", "..", "scripts", "testing", "docker_registry_image_file_permissions")), "\\", "\\\\")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testDockerRegistryImageDockerfileOutsideContext"), pushOptions.Registry, pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.outside_context", "sha256_digest"),
				),
			},
		},
	})
}

func TestAccDockerRegistryImageResource_pushMissingImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      loadTestConfiguration(t, RESOURCE, "docker_registry_image", "testDockerRegistryImagePushMissingConfig"),
				ExpectError: regexp.MustCompile("An image does not exist locally"),
			},
		},
	})
}

func testDockerRegistryImageNotInRegistry(pushOpts internalPushImageOptions) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		providerConfig := testAccProvider.Meta().(*ProviderConfig)
		username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
		digest, _ := getImageDigestWithFallback(pushOpts, username, password, true)
		if digest != "" {
			return fmt.Errorf("image found")
		}
		return nil
	}
}

func testDockerRegistryImageInRegistry(username, password string, pushOpts internalPushImageOptions, cleanup bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		digest, err := getImageDigestWithFallback(pushOpts, username, password, true)
		if err != nil || len(digest) < 1 {
			return fmt.Errorf("image '%s' with credentials('%s' - '%s') not found: %w", pushOpts.Name, username, password, err)
		}
		if cleanup {
			err := deleteDockerRegistryImage(pushOpts, digest, username, password, true, false)
			if err != nil {
				return fmt.Errorf("Unable to remove test image '%s': %w", pushOpts.Name, err)
			}
		}
		return nil
	}
}
