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

HUB ?= docker.io/apache
APP_NAME = skywalking-mcp

.PHONY: all
all: build ;

.PHONY: build
build: ## Build the binary.
	${MKDIR_P} bin/
	CGO_ENABLED=0 go build -ldflags "\
	    -X ${VERSION_PATH}.version=${VERSION} \
		-X ${VERSION_PATH}.commit=${GIT_COMMIT} \
		-X ${VERSION_PATH}.date=${BUILD_DATE}" \
		-o bin/swmcp cmd/skywalking-mcp/main.go

.PHONY: build-image
build-image: ## Build the Docker image.
	docker build -t skywalking-mcp:latest .

$(GO_LINT):
	@$(GO_LINT) version > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.0
$(LICENSE_EYE):
	@$(LICENSE_EYE) --version > /dev/null 2>&1 || go install github.com/apache/skywalking-eyes/cmd/license-eye@latest
	
.PHONY: lint
lint: $(GO_LINT)
	$(GO_LINT) run -v --timeout 5m ./...
.PHONY: fix-lint
fix-lint: $(GO_LINT)
	$(GO_LINT) run -v --fix ./...

.PHONY: license-header
license-header: clean $(LICENSE_EYE)
	@$(LICENSE_EYE) header check

.PHONY: fix-license-header
fix-license-header: clean $(LICENSE_EYE)
	@$(LICENSE_EYE) header fix

.PHONY: dependency-license
dependency-license: clean $(LICENSE_EYE)
	@$(LICENSE_EYE) dependency resolve --summary ./dist/LICENSE.tpl --output ./dist/licenses || exit 1
	@if [ ! -z "`git diff -U0 ./dist`" ]; then \
		echo "LICENSE file is not updated correctly"; \
		git diff -U0 ./dist; \
		exit 1; \
	fi

.PHONY: fix-dependency-license
fix-dependency-license: clean $(LICENSE_EYE)
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
	-rm -rf *.tgz
	-rm -rf *.asc
	-rm -rf *.sha512
	@go mod tidy &> /dev/null

.PHONY: docker
docker: PUSH_OR_LOAD = --load
docker: PLATFORMS =

.PHONY: docker.push
docker.push: PUSH_OR_LOAD = --push
docker.push: PLATFORMS = --platform linux/386,linux/amd64,linux/arm64

docker docker.push:
	docker buildx create --use --driver docker-container --name skywalking_mcp > /dev/null 2>&1 || true
	docker buildx build $(PUSH_OR_LOAD) $(PLATFORMS) --build-arg VERSION=$(VERSION) . -t $(HUB)/$(APP_NAME):$(VERSION) -t $(HUB)/$(APP_NAME):latest
	docker buildx rm skywalking_mcp