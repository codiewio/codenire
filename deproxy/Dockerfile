FROM ubuntu/squid

COPY squid.conf /etc/squid/squid.conf

COPY init.sh /usr/local/bin/init.sh
RUN chmod +x /usr/local/bin/init.sh
RUN /usr/local/bin/init.sh

VOLUME /var/spool/squid

EXPOSE 3128
