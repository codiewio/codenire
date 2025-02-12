FROM alpine:latest

RUN apk update && apk add --no-cache postgresql-client

RUN mkdir -p /app
WORKDIR /app

COPY run-sql.sh /app/run-sql.sh
RUN chmod +x /app/run-sql.sh