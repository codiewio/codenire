FROM php:8.4-cli-alpine

COPY php.ini /usr/local/etc/php/
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
