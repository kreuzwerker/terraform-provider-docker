package provider

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var registryDigestRegexp = regexp.MustCompile(`\A[A-Za-z0-9_\+\.-]+:[A-Fa-f0-9]+\z`)

func TestAccDockerRegistryImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_registry_image", "testAccDockerImageDataSourceConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_image.foo", "sha256_digest", registryDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerRegistryImage_private(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: loadTestConfiguration(t, DATA_SOURCE, "docker_registry_image", "testAccDockerImageDataSourcePrivateConfig"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_image.bar", "sha256_digest", registryDigestRegexp),
				),
			},
		},
	})
}

func TestAccDockerRegistryImage_auth(t *testing.T) {
	registry := "127.0.0.1:15000"
	image := "127.0.0.1:15000/tftest-service:v1"
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, DATA_SOURCE, "docker_registry_image", "testAccDockerImageDataSourceAuthConfig"), registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_image.foobar", "sha256_digest", registryDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestAccDockerRegistryImage_httpAuth(t *testing.T) {
	registry := "http://127.0.0.1:15001"
	image := "127.0.0.1:15001/tftest-service:v1"
	ctx := context.Background()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(loadTestConfiguration(t, DATA_SOURCE, "docker_registry_image", "testAccDockerImageDataSourceAuthConfig"), registry, image),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.docker_registry_image.foobar", "sha256_digest", registryDigestRegexp),
				),
			},
		},
		CheckDestroy: func(state *terraform.State) error {
			return checkAndRemoveImages(ctx, state)
		},
	})
}

func TestGetDigestFromResponse(t *testing.T) {
	headerContent := "sha256:2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"
	respWithHeaders := &http.Response{
		Header: http.Header{
			"Docker-Content-Digest": []string{headerContent},
		},
		Body: ioutil.NopCloser(bytes.NewReader([]byte("foo"))),
	}

	if digest, _ := getDigestFromResponse(respWithHeaders); digest != headerContent {
		t.Errorf("Expected digest from header to be %s, but was %s", headerContent, digest)
	}

	bodyDigest := "sha256:fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9"
	respWithoutHeaders := &http.Response{
		Header: make(http.Header),
		Body:   ioutil.NopCloser(bytes.NewReader([]byte("bar"))),
	}

	if digest, _ := getDigestFromResponse(respWithoutHeaders); digest != bodyDigest {
		t.Errorf("Expected digest calculated from body to be %s, but was %s", bodyDigest, digest)
	}
}
