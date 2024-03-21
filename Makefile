TAG := $(shell git rev-parse --short HEAD)
DIR := $(shell pwd -L)
DIR := $(shell pwd -L)
SDCLI_VERSION :=v1.5
SDCLI=docker run --rm -v "$(DIR):$(DIR)" -w "$(DIR)" asecurityteam/sdcli:$(SDCLI_VERSION)


dep:
	$(SDCLI) go dep

lint:
	$(SDCLI) go lint

test:
	$(SDCLI) go test

integration:
	$(SDCLI) go integration

coverage:
	$(SDCLI) go coverage

doc: ;

build-dev: ;

build: ;

run: ;

deploy-dev: ;

deploy: ;
