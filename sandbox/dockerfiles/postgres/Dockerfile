FROM alpine:latest

RUN apk update && apk add --no-cache postgresql-client

RUN mkdir -p /app
WORKDIR /app

RUN mkdir -p /run
COPY run-sql.sh /run/run-sql.sh
RUN chmod +x /run/run-sql.sh