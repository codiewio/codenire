FROM alpine:latest

RUN apk update && apk add --no-cache postgresql-client
RUN apk add --no-cache bash=5.2.15-r0

RUN mkdir -p /app
WORKDIR /app

RUN mkdir -p /run
COPY run-sql.sh /run/run-sql.sh
RUN chmod +x /run/run-sql.sh