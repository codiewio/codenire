#!/bin/bash

set -e

# Disable interactive apt prompts
export DEBIAN_FRONTEND=noninteractive

# https://stackoverflow.com/questions/54327058/aws-ami-need-to-explicitly-remove-apt-locks-when-provisioning-ami-from-bionic
while [ ! -f /var/lib/cloud/instance/boot-finished ]; do
  echo "Waiting for cloud init ..."
  sleep 5
done

while fuser /var/lib/apt/lists/lock >/dev/null 2>&1 ; do
  echo "Waiting for cloud init ..."
  sleep 5
done

# sudo rm -r /var/lib/apt/lists/*

cd /ops

# Dependencies
sudo apt-get update
# https://superuser.com/questions/1412054/non-interactive-apt-upgrade
# https://serverfault.com/questions/48724/100-non-interactive-debian-dist-upgrade
apt-get \
	-o Dpkg::Options::=--force-confold \
	-o Dpkg::Options::=--force-confdef \
	-y --allow-downgrades \
	--allow-remove-essential \
	--allow-change-held-packages \
	dist-upgrade
# sudo apt-get -y dist-upgrade
sudo apt-get -y upgrade
sudo apt-get -y autoremove
sudo apt-get install -y unzip tree jq curl tmux software-properties-common make

# Disable the firewall
sudo ufw disable || echo "ufw not installed"

# Docker
distro=$(lsb_release -si | tr '[:upper:]' '[:lower:]')
sudo apt-get install -y apt-transport-https ca-certificates gnupg2
curl -fsSL https://download.docker.com/linux/debian/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/${distro} $(lsb_release -cs) stable"
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin

# TODO:: remove
sudo apt-get install -y docker-compose-plugin

