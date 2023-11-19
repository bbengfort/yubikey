# Dynamic Builds
ARG BUILDER_IMAGE=golang:1.21-bookworm
ARG FINAL_IMAGE=debian:bookworm-slim

# Build stage
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Build Args
ARG GIT_REVISION=""

# Ensure ca-certificates are up to date on the image
RUN update-ca-certificates

# Use modules for dependencies
WORKDIR $GOPATH/src/github.com/bbengfort/yubikey

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=1
ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

# Copy package
COPY . .

# Build binary
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o /go/bin/yubikey -ldflags="-X 'github.com/bbengfort/yubikey.GitVersion=${GIT_REVISION}'" ./cmd/yubikey

# Final Stage
FROM --platform=${BUILDPLATFORM} ${FINAL_IMAGE} AS final

LABEL maintainer="Benjamin Bengfort <benjamin@rotational.io>"
LABEL description="Yubikey authn testing and registration service"

# Ensure ca-certificates are up to date
RUN set -x && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/yubikey /usr/local/bin/yubikey

CMD [ "/usr/local/bin/yubikey", "serve" ]