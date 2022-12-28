TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
PKG_NAME=internal/provider

GOLANGCI_VERSION = 1.49.0

# Values to install the provider locally for testing purposes
HOSTNAME=registry.terraform.io
NAMESPACE=kreuzwerker
NAME=docker
BINARY=terraform-provider-${NAME}
VERSION=9.9.9
OS_ARCH=$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)

.PHONY: build test testacc fmt fmtcheck test-compile website-link-check website-lint website-lint-fix

default: build

build: fmtcheck
	go install

local-build:
	go build -o ${BINARY}

local-install: local-build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

setup:
	go install github.com/katbyte/terrafmt
	go install github.com/client9/misspell/cmd/misspell
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
	rm -f .git/hooks/commit-msg \
	&& curl --fail -o .git/hooks/commit-msg https://raw.githubusercontent.com/hazcod/semantic-commit-hook/master/commit-msg \
	&& chmod 500 .git/hooks/commit-msg


bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint

bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | BINARY=golangci-lint bash -s -- v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

golangci-lint: bin/golangci-lint
	@bin/golangci-lint run ./...

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

fmt:
	gofmt -s -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

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

chlog-%:
	@echo "Generating CHANGELOG.md"
	git-chglog --next-tag $* -o CHANGELOG.md
	@echo "Version updated to $*!"
	@echo
	@echo "Review the changes made by this script then execute the following:"


replace-occurences-%:
	@echo "Replace occurences of old version strings..."
	sed -i '' "s/$(shell (svu --strip-prefix current))/$*/g" README.md docs/index.md examples/provider/provider-tf12.tf examples/provider/provider-tf13.tf

release-%:
	@echo "Review the changes made by this script then execute the following:"
	@${MAKE} website-generation
	@echo "Review the changes made by this script then execute the following:"
	@echo
	@echo "git add CHANGELOG.md README.md docs/index.md examples/provider/provider-tf12.tf examples/provider/provider-tf13.tf && git commit -m 'chore: Prepare release $*' && git tag -m 'Release $*' ${TAG_PREFIX}$*"
	@echo
	@echo "Finally, push the changes:"
	@echo
	@echo "git push; git push origin ${TAG_PREFIX}$*"

patch:
	@${MAKE} chlog-$(shell (svu patch))
	@${MAKE} replace-occurences-$(shell (svu --strip-prefix patch))
	@${MAKE} release-$(shell (svu patch))

minor:
	@${MAKE} chlog-$(shell (svu minor))
	@${MAKE} replace-occurences-$(shell (svu --strip-prefix minor))
	@${MAKE} release-$(shell (svu minor))

major:
	@${MAKE} chlog-$(shell (svu major))
	@${MAKE} replace-occurences-$(shell (svu --strip-prefix major))
	@${MAKE} release-$(shell (svu major))
