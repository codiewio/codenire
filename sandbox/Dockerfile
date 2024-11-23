# This is the sandbox backend server.
#
# When it's run, the host maps in /var/run/docker.sock to this
# environment so the play-sandbox server can connect to the host's
# docker daemon, which has the gvisor "runsc" runtime available.

FROM golang:1.23.3 AS build

ADD go.mod go.sum /m/
RUN cd /m && go mod download


# Intermediate stage: Build the binary
FROM golang:1.23.3 as builder

COPY --from=modules /go/pkg /go/pkg

# add a non-privileged user
RUN useradd -u 10001 app-user

RUN mkdir -p /app
COPY . /app
WORKDIR /app

# Build the binary with go build
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags prod -o ./bin/sandbox

FROM ubuntu:latest

RUN apt-get update

# Extra stuff for occasional debugging:
RUN apt-get install --yes strace lsof emacs-nox net-tools tcpdump procps

# Install Docker CLI:
RUN apt-get install --yes \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg2 \
        software-properties-common

RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
RUN apt-get update && apt-get install -y docker-ce-cli

COPY --from=builder /app/bin/sandbox /usr/local/bin/play-sandbox

ENTRYPOINT ["/usr/local/bin/play-sandbox"]
