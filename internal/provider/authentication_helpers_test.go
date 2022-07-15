package provider

import "testing"

func TestIsECRRepositoryURL(t *testing.T) {

	if !isECRRepositoryURL("825095469820.dkr.ecr.eu-central-1.amazonaws.com") {
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
