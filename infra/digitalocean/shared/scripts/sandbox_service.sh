#!/bin/bash

set -e

FILE="/ops/config/plugin_builder_installer.sh"

if [ -f "$FILE" ]; then
  echo "File found."

  if [ ! -x "$FILE" ]; then
    echo "Execute permission is missing. Adding..."
    chmod +x "$FILE"
  else
    echo "Execute permission is already set."
  fi

  bash "$FILE"
else
  echo "File not found."
fi