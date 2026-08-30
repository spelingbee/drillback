# The image drillback ships for NAS users: drillback, the docker CLI, the compose
# plugin, and restic in one place, so that a box which has a docker daemon and
# nothing else can run a restore drill.
#
# It is a CLIENT image. It contains no docker daemon. It talks to the daemon on the
# host through a mounted socket, which is a serious grant: see docs/docker.md, which
# spells out exactly what that gives away and how to give away less.
#
# Build:
#   docker build -t drillback:dev .
# Run: see docs/docker.md. There is one non-obvious rule and it is not optional -
# the workspace directory must be mounted at the SAME path inside and outside the
# container, because the bind mounts drillback asks for are resolved by the daemon on
# the host, not inside this container.

FROM golang:1.27-alpine AS build
WORKDIR /src

# Cached separately from the source so that a source-only change does not re-download
# the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=0.0.0-docker
ARG COMMIT=unknown
ARG DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w \
        -X github.com/spelingbee/drillback/internal/cli.Version=${VERSION} \
        -X github.com/spelingbee/drillback/internal/cli.Commit=${COMMIT} \
        -X github.com/spelingbee/drillback/internal/cli.Date=${DATE}" \
      -o /out/drillback ./cmd/drillback

FROM alpine:3.22

# docker-cli-compose is the compose v2 plugin; drillback shells out to
# `docker compose`, not to the removed `docker-compose` binary.
RUN apk add --no-cache \
      ca-certificates \
      docker-cli \
      docker-cli-compose \
      restic \
      tzdata

# A non-root user by default. It cannot read the docker socket on most hosts, which
# is the point: mounting the socket is a decision the operator makes explicitly, and
# docs/docker.md tells them to pass --group-add for the socket's gid rather than to
# run this as root.
RUN addgroup -g 65532 -S drillback \
 && adduser  -u 65532 -S -G drillback -h /home/drillback drillback

COPY --from=build /out/drillback /usr/local/bin/drillback

# The workspace parent this image expects you to use. Mount a host directory here at
# the SAME path (-v /var/lib/drillback:/var/lib/drillback) and pass
# `--workspace /var/lib/drillback`, or the compose stack the daemon starts will be told
# to bind-mount paths that exist only inside this container. There is no environment
# variable for it: drillback reads --workspace and nothing else, and an ENV here that
# the binary ignores would be a lie in a file people copy from.
RUN mkdir -p /var/lib/drillback && chown drillback:drillback /var/lib/drillback

USER drillback
WORKDIR /home/drillback

LABEL org.opencontainers.image.title="drillback" \
      org.opencontainers.image.description="Prove a backup restores, by restoring it." \
      org.opencontainers.image.source="https://github.com/spelingbee/drillback" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/spelingbee/drillback/blob/main/docs/docker.md"

ENTRYPOINT ["/usr/local/bin/drillback"]
CMD ["--help"]
