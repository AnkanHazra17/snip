FROM postgres:17
COPY --chmod=0644 schema.sql          /docker-entrypoint-initdb.d/001-schema.sql
COPY --chmod=0755 002-replication.sh  /docker-entrypoint-initdb.d/002-replication.sh