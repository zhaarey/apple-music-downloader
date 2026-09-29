# syntax=docker/dockerfile:1
ARG GO_VERSION=1.23

# Cross-compile on the build host; the binary is pure Go, so no emulation is needed here.
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
ARG REPO_OWNER=zhaarey
ARG REPO_NAME=apple-music-downloader
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w \
      -X 'amdl/internal/version.Version=${VERSION}' \
      -X 'amdl/internal/version.Commit=${COMMIT}' \
      -X 'amdl/internal/version.BuildDate=${BUILD_DATE}' \
      -X 'amdl/internal/updater.RepoOwner=${REPO_OWNER}' \
      -X 'amdl/internal/updater.RepoName=${REPO_NAME}'" \
      -o /out/amdl .

FROM alpine:3.22
# ffmpeg is used for animated artwork and post-download conversion.
RUN apk add --no-cache ca-certificates ffmpeg tzdata
COPY --from=builder /out/amdl /usr/local/bin/amdl
# config.yaml is created here from the embedded template on first run, and the
# relative save folders (AM-Lossless, AM-Atmos, ...) resolve here as well.
WORKDIR /data
VOLUME ["/data"]
ENTRYPOINT ["/usr/local/bin/amdl"]
