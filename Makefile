TAG := $(shell git rev-parse --short HEAD)

dep:
	dep ensure

lint:
	golangci-lint run --config .golangci.yaml ./...

test:
	mkdir -p .coverage
	go test -v -cover -coverpkg=./... -coverprofile=.coverage/unit.cover.out ./...
	gocov convert .coverage/unit.cover.out | gocov-xml > .coverage/unit.xml

integration: ;

coverage:
	mkdir -p .coverage
	gocovmerge .coverage/*.cover.out > .coverage/combined.cover.out
	gocov convert .coverage/combined.cover.out | gocov-xml > .coverage/combined.xml

doc: ;

build-dev: ;

build: ;

run: ;

deploy-dev: ;

deploy: ;