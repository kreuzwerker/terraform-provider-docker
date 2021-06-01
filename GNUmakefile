TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=internal/provider

default: build

build: fmtcheck
	go install

setup:
	go mod download
	cd tools && GO111MODULE=on go install github.com/client9/misspell/cmd/misspell
	cd tools && GO111MODULE=on go install github.com/katbyte/terrafmt
	cd tools && GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint
	cd tools && GO111MODULE=on go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
	rm -f .git/hooks/commit-msg \
	&& curl --fail -o .git/hooks/commit-msg https://raw.githubusercontent.com/hazcod/semantic-commit-hook/master/commit-msg \
	&& chmod 500 .git/hooks/commit-msg

golangci-lint:
	@golangci-lint run ./...

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
	@sh -c "curl -sL https://git.io/goreleaser | bash -s -- --rm-dist --skip-publish --snapshot --skip-sign"

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

website-generation:
	go generate
	
website-link-check:
	@scripts/markdown-link-check.sh

website-lint:
	@echo "==> Checking website against linters..."
	@misspell -error -source=text docs/ || (echo; \
		echo "Unexpected mispelling found in website files."; \
		echo "To automatically fix the misspelling, run 'make website-lint-fix' and commit the changes."; \
		exit 1)
	@docker run --rm -v $(PWD):/markdown 06kellyjac/markdownlint-cli docs/ || (echo; \
		echo "Unexpected issues found in website Markdown files."; \
		echo "To apply any automatic fixes, run 'make website-lint-fix' and commit the changes."; \
		exit 1)
	@terrafmt diff ./docs --check --pattern '*.md' --quiet || (echo; \
		echo "Unexpected differences in website HCL formatting."; \
		echo "To see the full differences, run: terrafmt diff ./docs --pattern '*.md'"; \
		echo "To automatically fix the formatting, run 'make website-lint-fix' and commit the changes."; \
		exit 1)

website-lint-fix:
	@echo "==> Applying automatic website linter fixes..."
	@misspell -w -source=text docs/
	@docker run --rm -v $(PWD):/markdown 06kellyjac/markdownlint-cli --fix docs/
	@terrafmt fmt ./docs --pattern '*.md'

.PHONY: build test testacc vet fmt fmtcheck errcheck test-compile website-link-check website-lint website-lint-fix

