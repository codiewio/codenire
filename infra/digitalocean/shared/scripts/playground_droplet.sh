#!/bin/bash

set -e

sudo mkdir -p /letsencrypt
sudo touch /letsencrypt/acme.json
sudo chmod 600 /letsencrypt/acme.json

