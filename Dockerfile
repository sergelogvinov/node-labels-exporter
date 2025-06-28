# syntax = docker/dockerfile:1.16
########################################

FROM golang:1.24-bookworm AS develop

WORKDIR /src
COPY ["go.mod", "go.sum", "/src"]
RUN go mod download

########################################

FROM --platform=${BUILDPLATFORM} golang:1.24.4-alpine3.22 AS builder
RUN apk update && apk add --no-cache make
ENV GO111MODULE=on
WORKDIR /src

COPY ["go.mod", "go.sum", "/src"]
RUN go mod download && go mod verify

COPY . .
ARG TAG
ARG SHA
RUN make build-all-archs

########################################

FROM --platform=${TARGETARCH} scratch AS node-labels-exporter
LABEL org.opencontainers.image.source="https://github.com/sergelogvinov/node-labels-exporter" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.description="Node labels exporter"

COPY --from=gcr.io/distroless/static-debian12:nonroot . .
ARG TARGETARCH
COPY --from=builder /src/bin/node-labels-exporter-${TARGETARCH} /bin/node-labels-exporter

ENTRYPOINT ["/bin/node-labels-exporter"]
