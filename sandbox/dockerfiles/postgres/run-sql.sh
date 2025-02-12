#!/bin/sh
set -e

for file in /app/*.sql; do
  echo "$file"
  psql -f "$file"
done
