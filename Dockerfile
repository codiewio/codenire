# Initial stage: download modules
FROM golang:1.22 as modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

# Intermediate stage: Build the binary
FROM golang:1.22 as builder

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
    -o ./bin/plugin . \

# Build the binary with go build
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags prod -o ./bin/playground .

# Final stage: Run the binary
FROM scratch

# and finally the binary
COPY --from=builder /playground/bin/playground /playground

CMD ["/playground"]