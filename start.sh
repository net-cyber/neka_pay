set -e

echo "Starting the application..."

echo "run db migrations"
source app/app.env
/app/migrate -path /app/migrations -database "$DB_SOURCE" -verbose up

echo "start the application"

/app/app