# syntax=docker/dockerfile:1

# Build a local Go toolchain image
FROM golang:1.24 AS go
# Intentionally empty: this stage serves as a runnable Go toolchain container

# Build a local golangci-lint image
FROM golangci/golangci-lint:v2.6 AS lint
USER root
# Intentionally empty: this stage serves as a runnable golangci-lint container


