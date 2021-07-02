package provider

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDockerRegistryImageTags() *schema.Resource {
	return &schema.Resource{
		Description: "Reads the tags of an image from a Docker Registry.",
		ReadContext: dataSourceDockerRegistryImagesRead,
		Schema: map[string]*schema.Schema{
			"images": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image": {
							Type:        schema.TypeString,
							Description: "The name of an image in the repository",
							Computed:    true,
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
				},
				Computed:    true,
				Description: "The images and their tags available in the registry",
			},
		},
	}
}

// Need to make this resliant against 404s...
func dataSourceDockerRegistryImagesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	authConfig := meta.(*ProviderConfig).AuthConfigs

	var c types.AuthConfig
	if len(authConfig.Configs) == 1 {
		for _, config := range authConfig.Configs {
			c = config
		}
	}

	repositories, err := getRepositories(c.ServerAddress, c.Username, c.Password)
	if err != nil {
		return diag.Errorf("Got error when attempting to fetch image version from registry: %s", err)
	}

	var rr []repoMap
	shas := make([]string, len(repositories))
	for i, repository := range repositories {
		sha := repository.sha256()
		shas[i] = sha
		t, err := getTagsOfImage(c.ServerAddress, repository.image, c.Username, c.Password)
		if err != nil {
			return diag.Errorf("failed to get tag for %s: %w", repository.image, err)
		}
		rr = append(rr, repoMap{
			"image": repository.image,
			"tags":  t,
		})
	}

	allTags := strings.Join(shas, "")
	id := sha256.Sum256([]byte(allTags))

	d.SetId(fmt.Sprintf("%x", id))
	log.Println("[DEBUG] got these responses from AWS: ", spew.Sprint(repositories))

	err = d.Set("images", rr) // meh name!
	if err != nil {
		panic(err)
	}

	return nil
}

type repoMap map[string]interface{}

type Repo struct {
	image string
	tags  []string
}

func (r Repo) sha256() string {
	allTags := strings.Join(r.tags, "")
	id := sha256.Sum256([]byte(fmt.Sprintf("%s%s", r.image, allTags)))
	return fmt.Sprintf("%x", id)
}

func getRepositories(registry, username, password string) ([]Repo, error) {
	log.Println("[###############] fetching repos...")
	var repos []Repo
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

	req, err := http.NewRequest("GET", registry+"/v2/_catalog", nil)
	if err != nil {
		return repos, fmt.Errorf("Error creating registry request: %s", err)
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
		return repos, fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		log.Println("[###############] Got a positive reponse from AWS")
		return getReposFrom(resp)

    case http.StatusNotFound:
        return []Repo{}, nil
	case http.StatusUnauthorized:
		authHeader := resp.Header.Get("www-authenticate")
		if strings.HasPrefix(authHeader, "Bearer") {

			token, err := getBearerToken(client, authHeader, username, password)
			if err != nil {
				return repos, err
			}

			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil {
				return repos, fmt.Errorf("Error during registry request: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				return repos, fmt.Errorf("Got bad response from registry: " + resp.Status)
			}

			return getReposFrom(resp)
		}

		return repos, fmt.Errorf("Bad credentials: " + resp.Status)

	default:
		return repos, fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}

func getReposFrom(response *http.Response) ([]Repo, error) {
	var repos []Repo
	r := struct {
		Repositories []string
	}{}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return []Repo{}, err
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return []Repo{}, err
	}

	for _, reposity := range r.Repositories {
		repos = append(repos, Repo{image: reposity, tags: []string{}})
	}

	return repos, nil
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

func getTagsOfImage(registry, image, username, password string) ([]string, error) {
	log.Println("[###############] fetching tags...")
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

	req, err := http.NewRequest("GET", registry+"/v2/"+image+"/tags/list", nil)
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
		return tags, fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		log.Println("[###############] Got a positive reponse from AWS")
		return getTagsFrom(resp)

	case http.StatusUnauthorized:
		authHeader := resp.Header.Get("www-authenticate")
		if strings.HasPrefix(authHeader, "Bearer") {

			token, err := getBearerToken(client, authHeader, username, password)
			if err != nil {
				return tags, err
			}

			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil {
				return tags, fmt.Errorf("Error during registry request: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				return tags, fmt.Errorf("Got bad response from registry: " + resp.Status)
			}

			return getTagsFrom(resp)
		}

		return tags, fmt.Errorf("Bad credentials: " + resp.Status)

	default:
		return tags, fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}
func getTagsFrom(response *http.Response) ([]string, error) {
	var noTags []string
	r := struct {
		Tags []string
	}{}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return noTags, err
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return noTags, err
	}

	return r.Tags, nil
}
