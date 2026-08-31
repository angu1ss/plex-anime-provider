FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/plex-anime-provider ./cmd/plex-anime-provider

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/plex-anime-provider /plex-anime-provider

# Inside the container we must listen on all interfaces; keep the port
# private by publishing it to 127.0.0.1 on the host (docker-compose.yml).
ENV PLEX_ANIME_PROVIDER_LISTEN=0.0.0.0:26463
EXPOSE 26463

# Distroless has no shell or curl: the binary probes its own /readyz.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/plex-anime-provider", "--healthcheck"]

ENTRYPOINT ["/plex-anime-provider"]
