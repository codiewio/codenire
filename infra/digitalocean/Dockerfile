FROM ubuntu:20.04

# Update
RUN apt-get -y update \
	&& apt-get -y dist-upgrade

# Install some prerequisites and utils
RUN apt-get -y install \
    curl \
    gnupg2 \
    wget \
    lsb-release \
    software-properties-common \
    vim \
    openssh-client \
    jq

# Install terraform, packer
RUN wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/hashicorp.list
RUN apt-get -y update && apt-get -y install terraform packer

COPY docker-entry.sh /scripts/docker-entry.sh
RUN chmod +x /scripts/docker-entry.sh
CMD ["/scripts/docker-entry.sh"]
