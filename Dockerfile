# syntax=docker/dockerfile:1

# Build a local Go toolchain image
FROM golang:1.27 AS go
USER root
# Intentionally empty: this stage serves as a runnable Go toolchain container


