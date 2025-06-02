package provider

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ECR HTTP authentication needs Bearer token in the Authorization header
// This token is an JWT which was b64 encoded again
// Depending on the aws cli command, the returned token is different
// "aws ecr get-login-password" is a simply JWT, which needs to be prefixed with "AWS:" and then b64 encoded
// "aws ecr get-authorization-token" is the best case, everything is encoded properly
// in case someone passes an base64 decoded token from "aws ecr get-authorization-token" we need to b64 encode it again
func normalizeECRPasswordForHTTPUsage(password string) string {
	if strings.HasPrefix(password, "ey") {
		return b64.StdEncoding.EncodeToString([]byte("AWS:" + password))
	} else if strings.HasPrefix(password, "AWS:") {
		return b64.StdEncoding.EncodeToString([]byte(password))
	}
	return password
}

// Docker operations need a JWT, so this function basically does the opposite as `normalizeECRPasswordForHTTPUsage`
// aws ecr get-authorization-token does not return a JWT, but a base64 encoded string which we need to decode
func normalizeECRPasswordForDockerCLIUsage(password string) string {
	if strings.HasPrefix(password, "ey") {
		return password
	}

	if !strings.HasPrefix(password, "AWS:") {
		decodedPassword, err := b64.StdEncoding.DecodeString(password)
		if err != nil {
			log.Fatalf("Error creating registry request: %s", err)
		}
		return string(decodedPassword)
	}

	return password[4:]
}

func isECRPublicRepositoryURL(url string) bool {
	return url == "public.ecr.aws"
}

func isECRRepositoryURL(url string) bool {
	if isECRPublicRepositoryURL(url) {
		return true
	}
	// Regexp is based on the ecr urls shown in https://docs.aws.amazon.com/AmazonECR/latest/userguide/registry_auth.html
	var ecrRexp = regexp.MustCompile(`^.*?dkr\.ecr\..*?\.amazonaws\.com$`)
	return ecrRexp.MatchString(url)
}

func isAzureCRRepositoryURL(url string) bool {
	// Regexp is based on the azurecr urls shown https://docs.microsoft.com/en-us/azure/container-registry/container-registry-get-started-portal?tabs=azure-cli#push-image-to-registry
	var azurecrRexp = regexp.MustCompile(`^.*\.azurecr\.io$`)
	return azurecrRexp.MatchString(url)
}

func setupHTTPHeadersForRegistryRequests(req *http.Request, fallback bool) {
	// We accept schema v2 manifests and manifest lists, and also OCI types
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	if fallback {
		// Fallback to this header if the registry does not support the v2 manifest like gcr.io
		req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v1+prettyjws")
	}
}

func setupHTTPRequestForRegistry(method, registry, registryWithProtocol, image, tag, username, password string, fallback bool) (*http.Request, error) {
	req, err := http.NewRequest(method, registryWithProtocol+"/v2/"+image+"/manifests/"+tag, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating registry request: %s", err)
	}

	if username != "" {
		if registry != "ghcr.io" && !isECRRepositoryURL(registry) && !isAzureCRRepositoryURL(registry) && registry != "gcr.io" {
			req.SetBasicAuth(username, password)
		} else {
			if isECRPublicRepositoryURL(registry) {
				password = normalizeECRPasswordForHTTPUsage(password)
				req.Header.Add("Authorization", "Bearer "+password)
			} else if isECRRepositoryURL(registry) {
				password = normalizeECRPasswordForHTTPUsage(password)
				req.Header.Add("Authorization", "Basic "+password)
			} else {
				req.Header.Add("Authorization", "Bearer "+b64.StdEncoding.EncodeToString([]byte(password)))
			}
		}
	}

	setupHTTPHeadersForRegistryRequests(req, fallback)

	return req, nil
}

// Parses key/value pairs from a WWW-Authenticate header
func parseAuthHeader(header string) (map[string]string, error) {
	if !strings.HasPrefix(header, "Bearer") {
		return nil, errors.New("missing or invalid www-authenticate header, does not start with 'Bearer'")
	}

	parts := strings.SplitN(header, " ", 2)
	parts = regexp.MustCompile(`\w+\=\".*?\"|\w+[^\s\"]+?`).FindAllString(parts[1], -1) // expression to match auth headers.
	opts := make(map[string]string)

	for _, part := range parts {
		vals := strings.SplitN(part, "=", 2)
		key := vals[0]
		val := strings.Trim(vals[1], "\", ")
		opts[key] = val
	}

	return opts, nil
}

func getAuthToken(auth map[string]string, username string, password string, fallbackScope string, client *http.Client) (string, error) {
	params := url.Values{}
	params.Set("service", auth["service"])
	params.Set("scope", auth["scope"])
	if auth["scope"] == "" {
		params.Set("scope", fallbackScope)
	}
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

	body, err := io.ReadAll(tokenResponse.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %s", err)
	}

	token := &TokenResponse{}
	err = json.Unmarshal(body, token)
	if err != nil {
		return "", fmt.Errorf("Error parsing OAuth token response: %s", err)
	}

	if token.Token != "" {
		return token.Token, nil
	}

	if token.AccessToken != "" {
		return token.AccessToken, nil
	}

	return "", fmt.Errorf("Error unsupported OAuth response")
}

type TokenResponse struct {
	Token       string
	AccessToken string `json:"access_token"`
}

var AuthConfigSchema = &schema.Schema{
	Type:        schema.TypeList,
	Description: "Authentication configuration for the Docker registry. It is only used for this resource.",
	Optional:    true,
	MaxItems:    1,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Description: "The address of the Docker registry.",
				Required:    true,
			},
			"username": {
				Type:        schema.TypeString,
				Description: "The username for the Docker registry.",
				Required:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "The password for the Docker registry.",
				Required:    true,
				Sensitive:   true,
			},
		},
	},
}
