-- Docker entrypoint init script
-- Database is created by POSTGRES_DB env var, this runs as superuser
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
