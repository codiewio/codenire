FROM postgres:16

COPY postgres.conf /etc/postgresql/postgresql.conf.sample

COPY restrict.sql /docker-entrypoint-initdb.d/restrict.sql

EXPOSE 5432
