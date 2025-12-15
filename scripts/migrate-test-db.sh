#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status

# Ensure required environment variables are set
if [[ -z "$DATABASE_USERNAME" || -z "$DATABASE_PASSWORD" || -z "$DATABASE_HOST" || -z "$DATABASE_PORT" || -z "$MYSQL_TEST_DATABASE" ]]; then
  echo "❌ Missing required environment variables. Exiting."
  exit 1
fi

# Run the migration
echo "🚀 Running migrations..."
migrate -database "mysql://$DATABASE_USERNAME:$DATABASE_PASSWORD@tcp($DATABASE_HOST:$DATABASE_PORT)/$MYSQL_TEST_DATABASE" -path migrations up

echo "✅ Migration completed successfully!"
