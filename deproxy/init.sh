#!/bin/bash

ALLOW_HOSTS=${ALLOW_HOSTS:-""}

if [ -z "$ALLOW_HOSTS" ]; then
  ALLOW_HOSTS=""
fi

ALLOWED_DOMAINS=$(echo " $ALLOW_HOSTS" | tr ',' ' ')
CONFIG_FILE="/etc/squid/squid.conf"

sed -i "/acl allowed_sites dstdomain/ s/\(acl allowed_sites dstdomain\)/\1 $ALLOWED_DOMAINS/" "$CONFIG_FILE"

