FROM ubuntu/squid:latest

COPY squid.conf /etc/squid/squid.conf

COPY setup.sh /usr/local/bin/setup.sh
RUN chmod +x /usr/local/bin/setup.sh

VOLUME /var/spool/squid

EXPOSE 3128

ENTRYPOINT ["/usr/local/bin/setup.sh"]
