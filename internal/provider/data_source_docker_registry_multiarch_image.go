package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerRegistryMultiarchImage() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the image metadata for each manifest in a Docker multi-arch image from a Docker Registry.",

		ReadContext: dataSourceDockerRegistryMultiarchImageRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image, including any tags. e.g. `alpine:latest`",
				Required:    true,
			},

			"manifests": {
				Type:        schema.TypeSet,
				Description: "The metadata for each manifest in the image",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"media_type": {
							Type:        schema.TypeString,
							Description: "The media type of the manifest.",
							Computed:    true,
						},
						"sha256_digest": {
							Type:        schema.TypeString,
							Description: "The content digest of the manifest, as stored in the registry.",
							Computed:    true,
						},
						"architecture": {
							Type:        schema.TypeString,
							Description: "The platform architecture supported by the manifest.",
							Computed:    true,
						},
						"os": {
							Type:        schema.TypeString,
							Description: "The operating system supported by the manifest.",
							Computed:    true,
						},
					},
				},
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

func dataSourceDockerRegistryMultiarchImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	manifest, err := getImageManifest(pullOpts.Registry, authConfig.ServerAddress, pullOpts.Repository, pullOpts.Tag, authConfig.Username, authConfig.Password, insecureSkipVerify, false)
	if err != nil {
		manifest, err = getImageManifest(pullOpts.Registry, authConfig.ServerAddress, pullOpts.Repository, pullOpts.Tag, authConfig.Username, authConfig.Password, insecureSkipVerify, true)
		if err != nil {
			return diag.Errorf("Got error when attempting to fetch image version %s:%s from registry: %s", pullOpts.Repository, pullOpts.Tag, err)
		}
	}

	d.SetId(fmt.Sprintf("%s:%s", pullOpts.Repository, pullOpts.Tag))
	if err = d.Set("manifests", flattenManifests(manifest.Manifests)); err != nil {
		log.Printf("[WARN] failed to set manifests from API: %s", err)
	}

	return nil
}

func getImageManifest(registry, registryWithProtocol, image, tag, username, password string, insecureSkipVerify, fallback bool) (*ManifestResponse, error) {
	client := buildHttpClientForRegistry(registryWithProtocol, insecureSkipVerify)

	req, err := setupHTTPRequestForRegistry("GET", registry, registryWithProtocol, image, tag, username, password, fallback)
	if err != nil {
		return nil, err
	}

	return doManifestRequest(req, client, username, password, true)
}

func doManifestRequest(req *http.Request, client *http.Client, username string, password string, retryUnauthorized bool) (*ManifestResponse, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		return getManifestsFromResponse(resp)

	default:
		if resp.StatusCode == http.StatusUnauthorized && retryUnauthorized {
			auth, err := parseAuthHeader(resp.Header.Get("www-authenticate"))
			if err != nil {
				return nil, fmt.Errorf("Bad credentials: %s", resp.Status)
			}

			token, err := getAuthToken(auth, username, password, client)
			if err != nil {
				return nil, err
			}

			req.Header.Set("Authorization", "Bearer "+token)

			return doManifestRequest(req, client, username, password, false)
		}

		return nil, fmt.Errorf("Got bad response from registry: %s", resp.Status)
	}
}

func getManifestsFromResponse(response *http.Response) (*ManifestResponse, error) {
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %s", err)
	}

	manifest := &ManifestResponse{}
	err = json.Unmarshal(body, manifest)
	if err != nil {
		return nil, fmt.Errorf("Error parsing manifest response: %s", err)
	}

	if len(manifest.Manifests) == 0 {
		return nil, fmt.Errorf("Error unsupported manifest response")
	}

	return manifest, nil
}

func flattenManifests(in []Manifest) []manifestMap {
	manifests := make([]manifestMap, len(in))
	for i, m := range in {
		manifests[i] = manifestMap{
			"media_type":    m.MediaType,
			"sha256_digest": m.Digest,
			"architecture":  m.Platform.Architecture,
			"os":            m.Platform.OS,
		}
	}

	return manifests
}

type ManifestResponse struct {
	Manifests []Manifest `json:"manifests"`
}

type Manifest struct {
	MediaType string           `json:"mediaType"`
	Digest    string           `json:"digest"`
	Platform  ManifestPlatform `json:"platform"`
}

type ManifestPlatform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

type manifestMap map[string]interface{}
