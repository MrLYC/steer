FROM node:20-alpine AS frontend-builder
ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV ALL_PROXY=
ENV NO_PROXY=
ENV http_proxy=
ENV https_proxy=
ENV all_proxy=
ENV no_proxy=
ENV npm_config_proxy=
ENV npm_config_https_proxy=
ENV npm_config_http_proxy=
ENV npm_config_no_proxy=
ENV NPM_CONFIG_PROXY=
ENV NPM_CONFIG_HTTPS_PROXY=
ENV NPM_CONFIG_HTTP_PROXY=
ENV NPM_CONFIG_NO_PROXY=
WORKDIR /app/ui
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN unset HTTP_PROXY HTTPS_PROXY ALL_PROXY NO_PROXY http_proxy https_proxy all_proxy no_proxy \
    && rm -f /root/.npmrc /etc/npmrc /.npmrc /root/.config/npm/npmrc || true \
    && rm -f /root/.config/pnpm/rc /root/.pnpmrc /.pnpmrc || true \
    && npm config delete proxy || true \
    && npm config delete https-proxy || true \
    && npm config delete http-proxy || true \
    && npm install -g pnpm \
    && pnpm config delete proxy || true \
    && pnpm config delete https-proxy || true \
    && pnpm config delete http-proxy || true \
    && pnpm install
COPY ui/ .
# NOTE: ui/package.json "build" also bundles server code into dist/ (esbuild)
# which overwrites the Vite SPA output (index.html/assets). For the operator's
# embedded web mode we only need the static SPA build.
RUN pnpm exec vite build

# Build the manager binary
FROM golang:1.21 AS builder
ARG TARGETOS
ARG TARGETARCH

# Some environments inject HTTP(S)_PROXY (e.g. pointing to 127.0.0.1) which
# breaks module downloads inside Docker build. Clear them explicitly.
ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV ALL_PROXY=
ENV NO_PROXY=
ENV http_proxy=
ENV https_proxy=
ENV all_proxy=
ENV no_proxy=

WORKDIR /workspace
# Copy the Go Modules manifests
COPY operator/go.mod go.mod
COPY operator/go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN unset HTTP_PROXY HTTPS_PROXY ALL_PROXY NO_PROXY http_proxy https_proxy all_proxy no_proxy \
    && go mod download

# Copy the go source
COPY operator/ .

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager ./cmd/main.go

# NOTE: Using alpine here to avoid pulling from gcr.io in environments where
# access is flaky (we previously hit timeouts). The manager binary is built
# with CGO_DISABLED=0 and works fine on alpine.
FROM alpine:3.19
ARG TARGETOS
ARG TARGETARCH
ARG HELM_VERSION=v3.14.4
WORKDIR /
COPY --from=builder /workspace/manager .
# Vite build output lands under dist/public
COPY --from=frontend-builder /app/ui/dist/public /static

# Install helm CLI for in-process helm operations (scheme A)
RUN apk add --no-cache ca-certificates curl tar \
    && OS="${TARGETOS:-linux}" \
    && case "${TARGETARCH:-amd64}" in \
      amd64|arm64) ARCH="${TARGETARCH:-amd64}" ;; \
      *) echo "unsupported TARGETARCH=${TARGETARCH}" && exit 1 ;; \
    esac \
    && curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-${OS}-${ARCH}.tar.gz" -o /tmp/helm.tgz \
    && tar -xzf /tmp/helm.tgz -C /tmp \
    && mv "/tmp/${OS}-${ARCH}/helm" /usr/local/bin/helm \
    && chmod +x /usr/local/bin/helm \
    && rm -rf /tmp/helm.tgz "/tmp/${OS}-${ARCH}"

RUN addgroup -S -g 65532 nonroot && adduser -S -D -H -u 65532 -G nonroot nonroot
USER 65532:65532

ENTRYPOINT ["/manager"]
