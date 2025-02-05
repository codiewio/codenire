FROM ubuntu/squid

LABEL container_name="package_dev"

COPY squid.conf /etc/squid/squid.conf

VOLUME /var/spool/squid

EXPOSE 3128

#CMD ["squid", "-N", "-d", "1"]