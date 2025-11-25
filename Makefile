TAG := $(shell git rev-parse --short HEAD)
DIR := $(shell pwd -L)
DIR := $(shell pwd -L)
DOCKERFILE ?= Dockerfile
LOCAL_GO_IMAGE ?= transport-go
LOCAL_LINT_IMAGE ?= transport-golangci-lint
GODOCKER = docker run --rm -v "$(DIR):$(DIR)" -w "$(DIR)" $(LOCAL_GO_IMAGE)
LINTDOCKER = docker run --rm -v "$(DIR):$(DIR)" -w "$(DIR)" $(LOCAL_LINT_IMAGE)

COVERAGE_DIR := .coverage
UNIT_COVERAGE_DIR := $(COVERAGE_DIR)/unit
UNIT_COVERAGE_FILE := $(UNIT_COVERAGE_DIR)/unit.cover.out

.PHONY: docker-build-go
docker-build-go:
	docker build --target go -t $(LOCAL_GO_IMAGE) -f $(DOCKERFILE) .

.PHONY: docker-build-lint
docker-build-lint:
	docker build --target lint -t $(LOCAL_LINT_IMAGE) -f $(DOCKERFILE) .

.PHONY: docker-build
docker-build: docker-build-go docker-build-lint

.PHONY: dep
dep: docker-build-go
	go mod vendor

.PHONY: lint
lint: docker-build-lint
	$(LINTDOCKER) golangci-lint run --config .golangci.yaml ./... -v

.PHONY: coverage-setup
coverage-setup:
	mkdir -p $(UNIT_COVERAGE_DIR)
	touch $(UNIT_COVERAGE_FILE)

.PHONY: test
test: coverage-setup docker-build-go
	$(GODOCKER) go test -coverprofile=$(UNIT_COVERAGE_FILE) -v -race ./...

integration: ;

.PHONY: coverage
coverage: docker-build-go
	$(GODOCKER) go tool cover -func=$(UNIT_COVERAGE_FILE)

doc: ;

build-dev: ;

build: ;

run: ;

deploy-dev: ;

deploy: ;
