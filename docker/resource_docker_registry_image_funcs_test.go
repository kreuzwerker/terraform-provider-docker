package docker

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-units"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestAccDockerRegistryImageResource_mapping(t *testing.T) {
	dummyProvider := Provider().(*schema.Provider)
	dummyResource := dummyProvider.ResourcesMap["docker_registry_image"]
	dummyResource.Create = func(d *schema.ResourceData, meta interface{}) error {
		build := d.Get("build").([]interface{})[0].(map[string]interface{})
		options := createImageBuildOptions(build)

		assert.Check(t, cmp.Equal(options.SuppressOutput, true))
		assert.Check(t, cmp.Equal(options.RemoteContext, "fooRemoteContext"))
		assert.Check(t, cmp.Equal(options.NoCache, true))
		assert.Check(t, cmp.Equal(options.Remove, true))
		assert.Check(t, cmp.Equal(options.ForceRemove, true))
		assert.Check(t, cmp.Equal(options.PullParent, true))
		assert.Check(t, cmp.Equal(options.Isolation, container.Isolation("hyperv")))
		assert.Check(t, cmp.Equal(options.CPUSetCPUs, "fooCpuSetCpus"))
		assert.Check(t, cmp.Equal(options.CPUSetMems, "fooCpuSetMems"))
		assert.Check(t, cmp.Equal(options.CPUShares, int64(4)))
		assert.Check(t, cmp.Equal(options.CPUQuota, int64(5)))
		assert.Check(t, cmp.Equal(options.CPUPeriod, int64(6)))
		assert.Check(t, cmp.Equal(options.Memory, int64(1)))
		assert.Check(t, cmp.Equal(options.MemorySwap, int64(2)))
		assert.Check(t, cmp.Equal(options.CgroupParent, "fooCgroupParent"))
		assert.Check(t, cmp.Equal(options.NetworkMode, "fooNetworkMode"))
		assert.Check(t, cmp.Equal(options.ShmSize, int64(3)))
		assert.Check(t, cmp.Equal(options.Dockerfile, "fooDockerfile"))
		assert.Check(t, cmp.Equal(len(options.Ulimits), 1))
		assert.Check(t, cmp.DeepEqual(*options.Ulimits[0], units.Ulimit{
			Name: "foo",
			Hard: int64(1),
			Soft: int64(2),
		}))
		assert.Check(t, cmp.Equal(len(options.BuildArgs), 1))
		assert.Check(t, cmp.Equal(*options.BuildArgs["HTTP_PROXY"], "http://10.20.30.2:1234"))
		assert.Check(t, cmp.Equal(len(options.AuthConfigs), 1))
		assert.Check(t, cmp.DeepEqual(options.AuthConfigs["foo.host"], types.AuthConfig{
			Username:      "fooUserName",
			Password:      "fooPassword",
			Auth:          "fooAuth",
			Email:         "fooEmail",
			ServerAddress: "fooServerAddress",
			IdentityToken: "fooIdentityToken",
			RegistryToken: "fooRegistryToken",
		}))
		assert.Check(t, cmp.DeepEqual(options.Labels, map[string]string{"foo": "bar"}))
		assert.Check(t, cmp.Equal(options.Squash, true))
		assert.Check(t, cmp.DeepEqual(options.CacheFrom, []string{"fooCacheFrom", "barCacheFrom"}))
		assert.Check(t, cmp.DeepEqual(options.SecurityOpt, []string{"fooSecurityOpt", "barSecurityOpt"}))
		assert.Check(t, cmp.DeepEqual(options.ExtraHosts, []string{"fooExtraHost", "barExtraHost"}))
		assert.Check(t, cmp.Equal(options.Target, "fooTarget"))
		assert.Check(t, cmp.Equal(options.SessionID, "fooSessionId"))
		assert.Check(t, cmp.Equal(options.Platform, "fooPlatform"))
		assert.Check(t, cmp.Equal(options.Version, types.BuilderVersion("1")))
		assert.Check(t, cmp.Equal(options.BuildID, "fooBuildId"))
		// output
		d.SetId("foo")
		d.Set("sha256_digest", "bar")
		return nil
	}
	dummyResource.Update = func(d *schema.ResourceData, meta interface{}) error {
		return nil
	}
	dummyResource.Delete = func(d *schema.ResourceData, meta interface{}) error {
		return nil
	}
	dummyResource.Read = func(d *schema.ResourceData, meta interface{}) error {
		d.Set("sha256_digest", "bar")
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: map[string]terraform.ResourceProvider{"docker": dummyProvider},
		Steps: []resource.TestStep{
			{
				Config: testBuildDockerRegistryImageMappingConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
	})

}

func TestAccDockerRegistryImageResource_build(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage:1.0")
	context := "../scripts/testing/docker_registry_image_context"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testDockerRegistryImageNotInRegistry(pushOptions),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testBuildDockerRegistryImageNoKeepConfig, pushOptions.Registry, pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
	})
}

func TestAccDockerRegistryImageResource_buildAndKeep(t *testing.T) {
	pushOptions := createPushImageOptions("127.0.0.1:15000/tftest-dockerregistryimage:1.0")
	context := "../scripts/testing/docker_registry_image_context"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testDockerRegistryImageInRegistry(pushOptions, true),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testBuildDockerRegistryImageKeepConfig, pushOptions.Registry, pushOptions.Name, context),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("docker_registry_image.foo", "sha256_digest"),
				),
			},
		},
	})
}

func TestAccDockerRegistryImageResource_pushMissingImage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testDockerRegistryImagePushMissingConfig,
				ExpectError: regexp.MustCompile("An image does not exist locally"),
			},
		},
	})
}

func testDockerRegistryImageNotInRegistry(pushOpts internalPushImageOptions) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		providerConfig := testAccProvider.Meta().(*ProviderConfig)
		username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
		digest, _ := getImageDigestWithFallback(pushOpts, username, password)
		if digest != "" {
			return fmt.Errorf("image found")
		}
		return nil
	}
}

func testDockerRegistryImageInRegistry(pushOpts internalPushImageOptions, cleanup bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		providerConfig := testAccProvider.Meta().(*ProviderConfig)
		username, password := getDockerRegistryImageRegistryUserNameAndPassword(pushOpts, providerConfig)
		digest, err := getImageDigestWithFallback(pushOpts, username, password)
		if err != nil || len(digest) < 1 {
			return fmt.Errorf("image not found")
		}
		if cleanup {
			err := deleteDockerRegistryImage(pushOpts, digest, username, password, false)
			if err != nil {
				return fmt.Errorf("Unable to remove test image. %s", err)
			}
		}
		return nil
	}
}

const testBuildDockerRegistryImageMappingConfig = `
resource "docker_registry_image" "foo" {
	name = "localhost:15000/foo:1.0"
	build {
		suppress_output = true
		remote_context = "fooRemoteContext"
		no_cache = true
		remove = true
		force_remove = true
		pull_parent = true
		isolation = "hyperv"
		cpu_set_cpus = "fooCpuSetCpus"
		cpu_set_mems = "fooCpuSetMems"
		cpu_shares = 4
		cpu_quota = 5
		cpu_period = 6
		memory = 1
		memory_swap = 2
		cgroup_parent = "fooCgroupParent"
		network_mode = "fooNetworkMode"
		shm_size = 3
		dockerfile = "fooDockerfile"
		ulimit {
			name = "foo"
			hard = 1
			soft = 2
		}
		auth_config {
			host_name = "foo.host"
			user_name = "fooUserName"
			password = "fooPassword"
			auth = "fooAuth"
			email = "fooEmail"
			server_address = "fooServerAddress"
			identity_token = "fooIdentityToken"
			registry_token = "fooRegistryToken"

		}
		build_args = {
			"HTTP_PROXY" = "http://10.20.30.2:1234"
		}
		context = "context"
		labels = {
			foo =  "bar"
		}
		squash = true
		cache_from = ["fooCacheFrom", "barCacheFrom"]
		security_opt = ["fooSecurityOpt", "barSecurityOpt"]
		extra_hosts = ["fooExtraHost", "barExtraHost"]
		target = "fooTarget"
		session_id = "fooSessionId"
		platform = "fooPlatform"
		version = "1"
		build_id = "fooBuildId"
	}
}
`

const testBuildDockerRegistryImageNoKeepConfig = `
provider "docker" {
	alias = "private"
	registry_auth {
		address  = 	"%s"
	}
}
resource "docker_registry_image" "foo" {
	provider = "docker.private"
	name = "%s"
	build {
		context = "%s"
		remove = true
		force_remove = true
		no_cache = true
	}
}
`

const testBuildDockerRegistryImageKeepConfig = `
provider "docker" {
	alias = "private"
	registry_auth {
		address  = 	"%s"
	}
}
resource "docker_registry_image" "foo" {
	provider = "docker.private"
	name = "%s"
	keep_remotely = true
	build {
		context = "%s"
		remove = true
		force_remove = true
		no_cache = true
	}
}
`

const testDockerRegistryImagePushMissingConfig = `
provider "docker" {
	alias = "private"
	registry_auth {
		address  = 	"127.0.0.1:15000"
	}
}
resource "docker_registry_image" "foo" {
	provider = "docker.private"
	name = "127.0.0.1:15000/nonexistent:1.0"
}
`
