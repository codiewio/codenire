#!/bin/bash

FILE="/ops/scripts/plugin_builder_installer.sh"

if [ -f "$FILE" ]; then
  echo "File found."

  if [ ! -x "$FILE" ]; then
    chmod +x "$FILE"
  fi

  bash "$FILE"

  exit 0
fi
