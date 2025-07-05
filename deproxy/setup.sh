#!/bin/bash
set -e

ALLOW_HOSTS=${ALLOW_HOSTS:-""}

if [ -z "$ALLOW_HOSTS" ]; then
  ALLOW_HOSTS=""
fi

ALLOWED_DOMAINS=$(echo " $ALLOW_HOSTS" | tr ',' ' ')
CONFIG_FILE="/etc/squid/squid.conf"

sed -i "/acl allowed_sites dstdomain/ s/\(acl allowed_sites dstdomain\)/\1 $ALLOWED_DOMAINS/" "$CONFIG_FILE"



ALLOW_ALL=${ALLOW_ALL:-false}
if [[ "$ALLOW_ALL" == "true" ]]; then
    sed -i '/^acl allowed_sites dstdomain /c\acl allowed_sites dstdom_regex .' "$CONFIG_FILE"
fi


# default behaviour is to launch squid
if [[ -z ${1} ]]; then
  if [[ ! -d ${SQUID_CACHE_DIR}/00 ]]; then
    echo "Initializing cache..."
    squid -N -f /etc/squid/squid.conf -z
  fi
  echo "Starting squid..."
  echo "Squid use config: "
  cat $CONFIG_FILE

  squid -f /etc/squid/squid.conf -NYCd 1
else
  exec "$@"
fi



