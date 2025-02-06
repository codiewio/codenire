FROM ubuntu/squid

ARG ALLOW_HOSTS
ENV ALLOW_HOSTS=${ALLOW_HOSTS}

COPY squid.conf /etc/squid/squid.conf

COPY setup.sh /usr/local/bin/setup.sh
RUN chmod +x /usr/local/bin/setup.sh
RUN /usr/local/bin/setup.sh

VOLUME /var/spool/squid

EXPOSE 3128
