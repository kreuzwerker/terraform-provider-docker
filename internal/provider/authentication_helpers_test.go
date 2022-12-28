package provider

import (
	"testing"
)

func TestIsECRRepositoryURL(t *testing.T) {

	if !isECRRepositoryURL("2385929435838.dkr.ecr.eu-central-1.amazonaws.com") {
		t.Fatalf("Expected true")
	}
	if !isECRRepositoryURL("39e39fmgvkgd.dkr.ecr.us-east-1.amazonaws.com") {
		t.Fatalf("Expected true")
	}
	if isECRRepositoryURL("39e39fmgvkgd.dkr.ecrus-east-1.amazonaws.com") {
		t.Fatalf("Expected false")
	}
	if !isECRRepositoryURL("public.ecr.aws") {
		t.Fatalf("Expected true")
	}
}

func TestParseAuthHeaders(t *testing.T) {
	header := "Bearer realm=\"https://gcr.io/v2/token\",service=\"gcr.io\",scope=\"repository:<owner>/:<repo>/<name>:pull\""
	result := parseAuthHeader(header)
	wantScope := "repository:<owner>/:<repo>/<name>:pull"
	if result["scope"] != wantScope {
		t.Errorf("want: %#v, got: %#v", wantScope, result["scope"])
	}

	header = "Bearer realm=\"https://gcr.io/v2/token\",service=\"gcr.io\",scope=\"repository:<owner>/:<repo>/<name>:push,pull\""
	result = parseAuthHeader(header)
	wantScope = "repository:<owner>/:<repo>/<name>:push,pull"
	if result["scope"] != wantScope {
		t.Errorf("want: %#v, got: %#v", wantScope, result["scope"])
	}
}
