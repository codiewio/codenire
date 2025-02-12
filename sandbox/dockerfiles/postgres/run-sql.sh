#!/bin/sh
set -e

for file in /app/*.sql; do
  psql \
    -c "\pset format wrapped" \
    -f "$file"
done
