package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerRegistryImage() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the image metadata from a Docker Registry. Used in conjunction with the [docker_image](../resources/image.md) resource to keep an image up to date on the latest available version of the tag.",

		ReadContext: dataSourceDockerRegistryImageRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image, including any tags. e.g. `alpine:latest`. You can also specify a digest, e.g. `nginx:1.28.0@sha256:eaa7e36decc3421fc04478c586dfea0d931cebe47d5bc0b15d758a32ba51126f`.",
				Required:    true,
			},

			"sha256_digest": {
				Type:        schema.TypeString,
				Description: "The content digest of the image, as stored in the registry.",
				Computed:    true,
			},

			"insecure_skip_verify": {
				Type:        schema.TypeBool,
				Description: "If `true`, the verification of TLS certificates of the server/registry is disabled. Defaults to `false`",
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func dataSourceDockerRegistryImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pullOpts := parseImageOptions(d.Get("name").(string))

	authConfig, err := getAuthConfigForRegistry(pullOpts.Registry, meta.(*ProviderConfig))
	if err != nil {
		// The user did not provide a credential for this registry.
		// But there are many registries where you can pull without a credential.
		// We are setting default values for the authConfig here.
		authConfig.Username = ""
		authConfig.Password = ""
		authConfig.ServerAddress = "https://" + pullOpts.Registry
	}

	insecureSkipVerify := d.Get("insecure_skip_verify").(bool)
	digest, err := getImageDigest(pullOpts.Registry, authConfig.ServerAddress, pullOpts.Repository, pullOpts.Tag, authConfig.Username, authConfig.Password, insecureSkipVerify, false)
	if err != nil {
		digest, err = getImageDigest(pullOpts.Registry, authConfig.ServerAddress, pullOpts.Repository, pullOpts.Tag, authConfig.Username, authConfig.Password, insecureSkipVerify, true)
		if err != nil {
			return diag.Errorf("Got error when attempting to fetch image version %s:%s from registry: %s", pullOpts.Repository, pullOpts.Tag, err)
		}
	}

	d.SetId(digest)
	d.Set("sha256_digest", digest)

	return nil
}

func getImageDigest(registry string, registryWithProtocol string, image, tag, username, password string, insecureSkipVerify, fallback bool) (string, error) {
	client := buildHttpClientForRegistry(registryWithProtocol, insecureSkipVerify)

	req, err := setupHTTPRequestForRegistry("HEAD", registry, registryWithProtocol, image, tag, username, password, fallback)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		return getDigestFromResponse(resp)

	// Either OAuth is required or the basic auth creds were invalid
	case http.StatusUnauthorized:
		auth, err := parseAuthHeader(resp.Header.Get("www-authenticate"))
		if err != nil {
			return "", fmt.Errorf("bad credentials: %s", resp.Status)
		}

		token, err := getAuthToken(auth, username, password, "repository:"+image+":push,pull", client)
		if err != nil {
			return "", err
		}

		req.Header.Set("Authorization", "Bearer "+token)

		// Do a HEAD request to docker registry first (avoiding Docker Hub rate limiting)
		digestResponse, err := doDigestRequest(req, client)
		if err != nil {
			return "", err
		}

		digest, err := getDigestFromResponse(digestResponse)
		if err == nil {
			return digest, nil
		}

		// If previous HEAD request does not contain required info, do a GET request
		req.Method = "GET"
		digestResponse, err = doDigestRequest(req, client)

		if err != nil {
			return "", err
		}

		return getDigestFromResponse(digestResponse)

	// Some unexpected status was given, return an error
	default:
		return "", fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}

func getDigestFromResponse(response *http.Response) (string, error) {
	header := response.Header.Get("Docker-Content-Digest")

	if header == "" {
		body, err := io.ReadAll(response.Body)
		if err != nil || len(body) == 0 {
			return "", fmt.Errorf("Error reading registry response body: %s", err)
		}

		return fmt.Sprintf("sha256:%x", sha256.Sum256(body)), nil
	}

	return header, nil
}

func doDigestRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	digestResponse, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during registry request: %s", err)
	}

	if digestResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Got bad response from registry: " + digestResponse.Status)
	}

	return digestResponse, nil
}
