#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

VERSION_PATH=main
VERSION ?= dev-$(shell git rev-parse --short HEAD)
GIT_COMMIT=$(shell git rev-parse HEAD)
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
MKDIR_P = mkdir -p

GO_LINT = golangci-lint
LICENSE_EYE = license-eye

HUB ?= ghcr.io/apache
APP_NAME = skywalking-mcp

.PHONY: all
all: build ;

.PHONY: help
help: ## Show this help message.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-25s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary.
	${MKDIR_P} bin/
	CGO_ENABLED=0 go build -ldflags "\
	  -X ${VERSION_PATH}.version=${VERSION} \
		-X ${VERSION_PATH}.commit=${GIT_COMMIT} \
		-X ${VERSION_PATH}.date=${BUILD_DATE}" \
		-o bin/swmcp cmd/skywalking-mcp/main.go

$(GO_LINT):
	@$(GO_LINT) version > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.0
$(LICENSE_EYE):
	@$(LICENSE_EYE) --version > /dev/null 2>&1 || go install github.com/apache/skywalking-eyes/cmd/license-eye@latest

lint: $(GO_LINT) ## Run linter.
	$(GO_LINT) run -v --timeout 5m ./...

fix-lint: $(GO_LINT) ## Auto-fix lint issues.
	$(GO_LINT) run -v --fix ./...

.PHONY: license-header
license-header: $(LICENSE_EYE)
	@$(LICENSE_EYE) header check

.PHONY: fix-license-header
fix-license-header: $(LICENSE_EYE)
	@$(LICENSE_EYE) header fix

.PHONY: dependency-license
dependency-license: $(LICENSE_EYE)
	@rm -rf ./dist/licenses
	@$(LICENSE_EYE) dependency resolve --summary ./dist/LICENSE.tpl --output ./dist/licenses || exit 1
	@if [ ! -z "`git diff -U0 ./dist`" ]; then \
		echo "LICENSE file is not updated correctly"; \
		git diff -U0 ./dist; \
		exit 1; \
	fi

.PHONY: fix-dependency-license
fix-dependency-license: $(LICENSE_EYE)
	@$(LICENSE_EYE) dependency resolve --summary ./dist/LICENSE.tpl --output ./dist/licenses

.PHONY: fix-license
fix-license: fix-license-header fix-dependency-license

.PHONY: fix
fix: fix-lint fix-license

.PHONY: clean
clean:
	-rm -rf bin
	-rm -rf coverage.txt
	-rm -rf *.tgz
	-rm -rf *.asc
	-rm -rf *.sha512

.PHONY: build-image
build-image: docker-build ## Build the Docker image.

.PHONY: docker-build
docker-build: ## Build the Docker image for linux/amd64 and linux/arm64 and push to the registry.
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t $(HUB)/$(APP_NAME):$(VERSION) \
		-t $(HUB)/$(APP_NAME):latest \
		--push .

.PHONY: docker-push
docker-push: docker-build ## Build and push multi-platform Docker images to the registry.

.PHONY: docker
docker: docker-build

.PHONY: docker.push
docker.push: docker-push

## Release
RELEASE_SCRIPTS := ./scripts/release.sh

release-binary: release-source ## Package binary archive
	${RELEASE_SCRIPTS} -b

release-source: ## Package source archive
	${RELEASE_SCRIPTS} -s

release-sign: ## Sign artifacts
	${RELEASE_SCRIPTS} -k mcp

release-assembly: release-binary release-sign ## Generate release package

PUSH_RELEASE_SCRIPTS := ./scripts/push-release.sh
release-push-candidate:
	${PUSH_RELEASE_SCRIPTS}

.PHONY: lint fix-lint
.PHONY: license-header fix-license-header dependency-license fix-dependency-license
.PHONY: release-binary release-source release-sign release-assembly
.PHONY: release-push-candidate
