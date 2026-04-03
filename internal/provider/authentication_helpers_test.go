package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestIsECRPublicRepositoryURL(t *testing.T) {
	if !isECRPublicRepositoryURL("public.ecr.aws") {
		t.Fatalf("Expected true")
	}
	if isECRPublicRepositoryURL("public.ecr.aws.com") {
		t.Fatalf("Expected false")
	}
}

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
	_, err := parseAuthHeader("")
	if err == nil || err.Error() != "missing or invalid www-authenticate header, does not start with 'Bearer'" {
		t.Fatalf("wanted \"missing or invalid www-authenticate header, does not start with 'Bearer'\", got nil")
	}

	header := "Bearer realm=\"https://gcr.io/v2/token\",service=\"gcr.io\",scope=\"repository:<owner>/:<repo>/<name>:pull\""
	result, err := parseAuthHeader(header)
	if err != nil {
		t.Errorf("wanted no error, got %s", err)
	}
	wantScope := "repository:<owner>/:<repo>/<name>:pull"
	if result["scope"] != wantScope {
		t.Errorf("want: %#v, got: %#v", wantScope, result["scope"])
	}

	header = "Bearer realm=\"https://gcr.io/v2/token\",service=\"gcr.io\",scope=\"repository:<owner>/:<repo>/<name>:push,pull\""
	result, err = parseAuthHeader(header)
	if err != nil {
		t.Errorf("wanted no error, got %s", err)
	}
	wantScope = "repository:<owner>/:<repo>/<name>:push,pull"
	if result["scope"] != wantScope {
		t.Errorf("want: %#v, got: %#v", wantScope, result["scope"])
	}
}

func TestParseAuthHeadersMalformed(t *testing.T) {
	_, err := parseAuthHeader("Bearer")
	if err == nil || err.Error() != "missing or invalid www-authenticate header parameters" {
		t.Fatalf("wanted malformed header parameters error, got %#v", err)
	}

	_, err = parseAuthHeader("Bearer realm")
	if err == nil || err.Error() != "missing or invalid www-authenticate key/value pair: realm" {
		t.Fatalf("wanted malformed key/value error, got %#v", err)
	}
}

func TestGetAuthTokenFallbackToAnonymousOnForbidden(t *testing.T) {
	var requests int32

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)

		if user, _, ok := r.BasicAuth(); ok && user != "" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"forbidden"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"anonymous-token"}`))
	}))
	defer tokenServer.Close()

	auth := map[string]string{
		"realm":   tokenServer.URL,
		"service": "registry.docker.io",
		"scope":   "repository:library/alpine:pull",
	}

	token, err := getAuthToken(auth, "user", "opaque-token", "repository:library/alpine:pull", tokenServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if token != "anonymous-token" {
		t.Fatalf("want token anonymous-token, got %s", token)
	}

	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("want 2 token requests (credentialed + anonymous fallback), got %d", got)
	}
}

func TestGetAuthTokenUsesFallbackScope(t *testing.T) {
	fallbackScope := "repository:library/alpine:pull"

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotScope := r.URL.Query().Get("scope"); gotScope != fallbackScope {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"message":"unexpected scope %s"}`, gotScope)))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"scoped-token"}`))
	}))
	defer tokenServer.Close()

	auth := map[string]string{
		"realm":   tokenServer.URL,
		"service": "registry.docker.io",
		"scope":   "",
	}

	token, err := getAuthToken(auth, "", "", fallbackScope, tokenServer.Client())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if token != "scoped-token" {
		t.Fatalf("want token scoped-token, got %s", token)
	}
}
