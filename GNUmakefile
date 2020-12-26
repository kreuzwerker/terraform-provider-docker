TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=docker

default: build

setup:
	rm -f .git/hooks/commit-msg \
	&& curl --fail -o .git/hooks/commit-msg \
	https://raw.githubusercontent.com/hazcod/semantic-commit-hook/master/commit-msg \
	&& chmod 500 .git/hooks/commit-msg

build: fmtcheck
	go install

test: fmtcheck
	go test -i $(TEST) || exit 1
	echo $(TEST) | \
		xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4

testacc_setup: fmtcheck
	@sh -c "'$(CURDIR)/scripts/testacc_setup.sh'"

testacc: fmtcheck
	@sh -c "'$(CURDIR)/scripts/testacc_full.sh'"

testacc_cleanup: fmtcheck
	@sh -c "'$(CURDIR)/scripts/testacc_cleanup.sh'"

compile: fmtcheck
	@sh -c "'$(CURDIR)/scripts/compile.sh'"

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -s -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"


test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

.PHONY: build test testacc vet fmt fmtcheck errcheck test-compile

