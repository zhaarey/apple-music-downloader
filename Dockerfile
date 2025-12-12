ARG GOVERSION=1.25.5

FROM --platform=$BUILDPLATFORM golang:${GOVERSION}-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /bin/apple-music-dl main.go

FROM sambaiz/mp4box
COPY --from=builder /bin/apple-music-dl /usr/local/bin/apple-music-dl
COPY config.yaml ./
ENTRYPOINT ["/usr/local/bin/apple-music-dl"]
