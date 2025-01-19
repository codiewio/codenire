#!/bin/bash

set -e

cd /ops

sudo mkdir /dockerfiles
sudo chmod 777 /dockerfiles

sudo chmod +r /ops/config/daemon.json
sudo cp /ops/config/daemon.json /etc/docker/daemon.json
sudo chmod 0644 /etc/docker/daemon.json

curl -L -o /var/lib/docker/runsc https://storage.googleapis.com/gvisor/releases/release/latest/x86_64/runsc
sudo chmod +x /var/lib/docker/runsc
sudo systemctl reload docker.service
