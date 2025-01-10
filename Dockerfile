# Initial stage: download modules
FROM golang:1.22 as modules

ADD go.mod go.sum /m/
RUN cd /m && go mod download

# Intermediate stage: Build the binary
FROM golang:1.22 as builder

COPY --from=modules /go/pkg /go/pkg

# add a non-privileged user
RUN useradd -u 10001 playground

RUN mkdir -p /playground
ADD . /playground
WORKDIR /playground

# Build the binary with go build
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags prod -o ./bin/playground .

# Final stage: Run the binary
FROM scratch

# don't forget /etc/passwd from previous stage
COPY --from=builder /etc/passwd /etc/passwd
USER playground

# and finally the binary
COPY --from=builder /playground/bin/playground /playground

CMD ["/playground"]