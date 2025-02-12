FROM postgres:16

# Копируем файл конфигурации в контейнер
COPY postgres.conf /etc/postgresql/postgresql.conf.sample

# Копируем SQL скрипт для ограничения доступа в контейнер
COPY restrict.sql /docker-entrypoint-initdb.d/restrict.sql

EXPOSE 5432
