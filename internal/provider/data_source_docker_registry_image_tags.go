package provider

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerRegistryImageTags() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the tags of an image from a Docker Registry.",

		ReadContext: dataSourceDockerRegistryImageTagsRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the Docker image, without tags. e.g. `alpine`",
				Required:    true,
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The tags of the image",
			},
		},
	}
}

func dataSourceDockerRegistryImageTagsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pullOpts := parseImageOptions(d.Get("name").(string))
	authConfig := meta.(*ProviderConfig).AuthConfigs

	// Use the official Docker Hub if a registry isn't specified
	if pullOpts.Registry == "" {
		pullOpts.Registry = "registry.hub.docker.com"
	} else {
		// Otherwise, filter the registry name out of the repo name
		pullOpts.Repository = strings.Replace(pullOpts.Repository, pullOpts.Registry+"/", "", 1)
	}

	if pullOpts.Registry == "registry.hub.docker.com" {
		// Docker prefixes 'library' to official images in the path; 'consul' becomes 'library/consul'
		if !strings.Contains(pullOpts.Repository, "/") {
			pullOpts.Repository = "library/" + pullOpts.Repository
		}
	}

	username := ""
	password := ""

	if auth, ok := authConfig.Configs[normalizeRegistryAddress(pullOpts.Registry)]; ok {
		username = auth.Username
		password = auth.Password
	}

	tags, err := getImageTags(pullOpts.Registry, pullOpts.Repository, username, password)
	if err != nil {
		return diag.Errorf("Got error when attempting to fetch image version from registry: %s", err)
	}

	sort.Strings(tags)
	allTags := strings.Join(tags, "")
	id := sha256.Sum256([]byte(allTags))

	d.SetId(fmt.Sprintf("%x", id))
	d.Set("tags", tags)

	return nil
}

func getImageTags(registry, image, username, password string) ([]string, error) {
	var tags []string
	client := http.DefaultClient

	// Allow insecure registries only for ACC tests
	// cuz we don't have a valid certs for this case
	if env, okEnv := os.LookupEnv("TF_ACC"); okEnv {
		if i, errConv := strconv.Atoi(env); errConv == nil && i >= 1 {
			// DevSkim: ignore DS440000
			cfg := &tls.Config{
				InsecureSkipVerify: true,
			}
			client.Transport = &http.Transport{
				TLSClientConfig: cfg,
			}
		}
	}

	req, err := http.NewRequest("GET", "https://"+registry+"/v2/"+image+"/tags/list", nil)
	if err != nil {
		return tags, fmt.Errorf("Error creating registry request: %s", err)
	}

	/* Can introduce some kind of filtering here if we wanted to:
	q := req.URL.Query()
	q.Add("n", "10")
	req.URL.RawQuery = q.Encode()
	*/

	if username != "" {
		req.SetBasicAuth(username, password)
	}

	// We accept schema v2 manifests and manifest lists, and also OCI types
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		return getTagsFromResponse(resp)

	case http.StatusUnauthorized:
		authHeader := resp.Header.Get("www-authenticate")
		if strings.HasPrefix(authHeader, "Bearer") {

			token, err := getBearerToken(client, authHeader, username, password)
			if err != nil {
				return []string{}, err
			}

			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil {
				return []string{}, fmt.Errorf("Error during registry request: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				return []string{}, fmt.Errorf("Got bad response from registry: " + resp.Status)
			}

			return getTagsFromResponse(resp)
		}

		return []string{}, fmt.Errorf("Bad credentials: " + resp.Status)

	default:
		return tags, fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}

func getTagsFromResponse(response *http.Response) ([]string, error) {
	r := struct {
		Tags []string
	}{}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return []string{}, err
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return []string{}, err
	}

	return r.Tags, nil
}

func getBearerToken(client *http.Client, authHeader, username, password string) (string, error) {
	auth := parseAuthHeader(authHeader)
	params := url.Values{}
	params.Set("service", auth["service"])
	params.Set("scope", auth["scope"])
	tokenRequest, err := http.NewRequest("GET", auth["realm"]+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("Error creating registry request: %s", err)
	}

	if username != "" {
		tokenRequest.SetBasicAuth(username, password)
	}

	tokenResponse, err := client.Do(tokenRequest)
	if err != nil {
		return "", fmt.Errorf("Error during registry request: %s", err)
	}

	if tokenResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Got bad response from registry: " + tokenResponse.Status)
	}

	body, err := ioutil.ReadAll(tokenResponse.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %s", err)
	}

	token := &TokenResponse{}
	err = json.Unmarshal(body, token)
	if err != nil {
		return "", fmt.Errorf("Error parsing OAuth token response: %s", err)
	}

	return token.Token, nil
}
