
#!/bin/bash
set -e

# Create a user that can only replicate
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  CREATE ROLE replicator WITH REPLICATION LOGIN PASSWORD '$REPLICATION_PASSWORD';
EOSQL

# Allow the replication user to connect
echo "host replication replicator all scram-sha-256" >> "$PGDATA/pg_hba.conf"
