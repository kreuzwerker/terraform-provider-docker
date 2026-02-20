package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/docker/docker/api/types/registry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerRegistryImageManifests() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the image metadata for each manifest in a Docker multi-arch image from a Docker Registry.",

		ReadContext: dataSourceDockerRegistryImageManifestsRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image, including any tags. e.g. `alpine:latest`",
				Required:    true,
			},

			"auth_config": AuthConfigSchema,

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

func dataSourceDockerRegistryImageManifestsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pullOpts := parseImageOptions(d.Get("name").(string))

	var authConfig registry.AuthConfig
	if v, ok := d.GetOk("auth_config"); ok {
		log.Printf("[INFO] Using auth config from resource: %s", v)
		authConfig = buildAuthConfigFromResource(v)
	} else {
		log.Printf("[INFO] Using auth config from provider: %s", v)
		var err error
		authConfig, err = getAuthConfigForRegistry(pullOpts.Registry, meta.(*ProviderConfig))
		if err != nil {
			// The user did not provide a credential for this registry.
			// But there are many registries where you can pull without a credential.
			// We are setting default values for the authConfig here.
			authConfig.Username = ""
			authConfig.Password = ""
			authConfig.ServerAddress = "https://" + pullOpts.Registry
		}
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

	return doManifestRequest(req, client, username, password, "repository:"+image+":push,pull", true)
}

func doManifestRequest(req *http.Request, client *http.Client, username string, password string, fallbackScope string, retryUnauthorized bool) (*ManifestResponse, error) {
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
				return nil, fmt.Errorf("bad credentials: %s", resp.Status)
			}

			token, err := getAuthToken(auth, username, password, fallbackScope, client)
			if err != nil {
				return nil, err
			}

			req.Header.Set("Authorization", "Bearer "+token)

			return doManifestRequest(req, client, username, password, fallbackScope, false)
		}

		return nil, fmt.Errorf("got bad response from registry: %s", resp.Status)
	}
}

func getManifestsFromResponse(response *http.Response) (*ManifestResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %s", err)
	}

	manifest := &ManifestResponse{}
	err = json.Unmarshal(body, manifest)
	if err != nil {
		return nil, fmt.Errorf("Error parsing manifest response: %s", err)
	}

	if len(manifest.Manifests) == 0 {
		log.Printf("[DEBUG] Manifest response was not for list: %s", string(body))
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
