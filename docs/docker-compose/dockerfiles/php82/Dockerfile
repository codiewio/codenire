FROM php:8.2-cli-alpine

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /app
