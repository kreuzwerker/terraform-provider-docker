package provider

import (
	b64 "encoding/base64"
	"log"
	"net/http"
	"regexp"
	"strings"
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

func isECRRepositoryURL(url string) bool {
	if url == "public.ecr.aws" {
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
