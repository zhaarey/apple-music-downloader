ARG GOVERSION=1.25.5

FROM --platform=$BUILDPLATFORM golang:${GOVERSION}-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /bin/apple-music-dl main.go

FROM gpac/ubuntu
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y --no-install-recommends ffmpeg && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/apple-music-dl /usr/local/bin/apple-music-dl
WORKDIR /app
COPY config.yaml ./
RUN echo 'alac-save-folder: "/downloads/ALAC"' >> config.yaml \
    && echo 'atmos-save-folder: "/downloads/Atmos"' >> config.yaml \
    && echo 'aac-save-folder: "/downloads/AAC"' >> config.yaml
ENTRYPOINT ["/usr/local/bin/apple-music-dl"]
