#!/bin/bash

set -e

if [[ -z "${DO_TOKEN}" ]]; then
	echo "DO_TOKEN env var not set. Exiting."
	exit 1
fi

# Default terraform do_token input variables to this token
echo "TF_VAR_do_token=$DO_TOKEN" >> /root/.bashrc
export TF_VAR_do_token=$DO_TOKEN
# Default packer do_token input variables to this token
echo "PKR_VAR_do_token=$DO_TOKEN" >> /root/.bashrc
export PKR_VAR_do_token=$DO_TOKEN


# Terraform User Api Key
if [[ -z "${TF_TOKEN}" ]]; then
	echo "TF_TOKEN env var not set (if you use Terraform Cloud state — add it in .env)."
fi

echo "TF_TOKEN_app_terraform_io=$TF_TOKEN" >> /root/.bashrc
export TF_TOKEN_app_terraform_io=$TF_TOKEN

# Traefik
if [[ -n "${LETSENCRYPT_EMAIL}" ]]; then
  export TF_VAR_letsencrypt_email=$LETSENCRYPT_EMAIL
  echo "TF_VAR_letsencrypt_email=$LETSENCRYPT_EMAIL" >> /root/.bashrc
fi

# domain
if [[ -n "${PLAYGROUND_DOMAIN}" ]]; then
  export TF_VAR_playground_domain=$PLAYGROUND_DOMAIN
  echo "TF_VAR_playground_domain=$PLAYGROUND_DOMAIN" >> /root/.bashrc
fi


#ID_RSA="/codenire-deploy/shared/ssh/id_rsa"
#ID_RSA_PUB="/codenire-deploy/shared/ssh/id_rsa.pub"
#
#if [ ! -f "$ID_RSA" ] && [ ! -f "$ID_RSA_PUB" ]; then
#    echo "SSH keys will be generated"
#    ssh-keygen -t ed25519 -q -N '' -f "$ID_RSA" -m PEM
#  else
#    echo "SSH keys already exists"
#fi
#
#
##PRIVATE_KEY=$(base64 -w 0 ./codenire-deploy/shared/ssh/id_rsa)
#PRIVATE_KEY=$(awk 'BEGIN {ORS="\\n"} {print}' /codenire-deploy/shared/ssh/id_rsa)
#PUBLIC_KEY=$(cat /codenire-deploy/shared/ssh/id_rsa.pub)
#
#echo "export TF_VAR_do_ssh_private_key='$PRIVATE_KEY'" >> /root/.bashrc
#export TF_VAR_do_ssh_private_key="$PRIVATE_KEY"
#
#echo "export TF_VAR_do_ssh_public_key='$PUBLIC_KEY'" >> /root/.bashrc
#export TF_VAR_do_ssh_public_key="$PUBLIC_KEY"

/bin/bash
