# The image restored ships for NAS users: restored, the docker CLI, the compose
# plugin, and restic in one place, so that a box which has a docker daemon and
# nothing else can run a restore drill.
#
# It is a CLIENT image. It contains no docker daemon. It talks to the daemon on the
# host through a mounted socket, which is a serious grant: see docs/docker.md, which
# spells out exactly what that gives away and how to give away less.
#
# Build:
#   docker build -t restored:dev .
# Run: see docs/docker.md. There is one non-obvious rule and it is not optional -
# the workspace directory must be mounted at the SAME path inside and outside the
# container, because the bind mounts restored asks for are resolved by the daemon on
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
        -X github.com/spelingbee/restored/internal/cli.Version=${VERSION} \
        -X github.com/spelingbee/restored/internal/cli.Commit=${COMMIT} \
        -X github.com/spelingbee/restored/internal/cli.Date=${DATE}" \
      -o /out/restored ./cmd/restored

FROM alpine:3.22

# docker-cli-compose is the compose v2 plugin; restored shells out to
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
RUN addgroup -g 65532 -S restored \
 && adduser  -u 65532 -S -G restored -h /home/restored restored

COPY --from=build /out/restored /usr/local/bin/restored

# The workspace parent this image expects you to use. Mount a host directory here at
# the SAME path (-v /var/lib/restored:/var/lib/restored) and pass
# `--workspace /var/lib/restored`, or the compose stack the daemon starts will be told
# to bind-mount paths that exist only inside this container. There is no environment
# variable for it: restored reads --workspace and nothing else, and an ENV here that
# the binary ignores would be a lie in a file people copy from.
RUN mkdir -p /var/lib/restored && chown restored:restored /var/lib/restored

USER restored
WORKDIR /home/restored

LABEL org.opencontainers.image.title="restored" \
      org.opencontainers.image.description="Prove a backup restores, by restoring it." \
      org.opencontainers.image.source="https://github.com/spelingbee/restored" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.documentation="https://github.com/spelingbee/restored/blob/main/docs/docker.md"

ENTRYPOINT ["/usr/local/bin/restored"]
CMD ["--help"]
