# Initial stage: download modules
FROM golang:1.23 as modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

# Intermediate stage: Build the binary
FROM golang:1.23 as builder

COPY --from=modules /go/pkg /go/pkg

RUN mkdir -p /playground
ADD . /playground
WORKDIR /playground

# Get the version name and git commit as a build argument
ARG GIT_VERSION
ARG GIT_COMMIT

# Get the operating system and architecture to build for
ARG TARGETOS
ARG TARGETARCH

RUN set -xe \
	&& GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 \
    go build \
    -tags prod \
    -o ./bin/playground .

# Final stage: Run the binary
FROM alpine:latest

RUN apk add --no-cache curl

# and finally the binary
COPY --from=builder /playground/bin/playground /playground
#COPY ide/dist/codenire/browser /static

CMD ["/playground"]

ARG LABEL_VERSION
ARG LABEL_COMMIT

LABEL org.opencontainers.image.description="The open-source sandbox based on Docker containers and Google gVisor."
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.revision="${LABEL_COMMIT}"
LABEL org.opencontainers.image.source="https://github.com/codiewio/codenire"
LABEL org.opencontainers.image.title="Codenire"
LABEL org.opencontainers.image.url="https://codenire.com"
#LABEL org.opencontainers.image.vendor="Codiew INC"
LABEL org.opencontainers.image.version="${LABEL_VERSION}"