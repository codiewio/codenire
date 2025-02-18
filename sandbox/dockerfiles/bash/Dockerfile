FROM alpine:latest

RUN apk update && apk add --no-cache postgresql-client
RUN apk add --no-cache bash

RUN mkdir -p /app
WORKDIR /app

