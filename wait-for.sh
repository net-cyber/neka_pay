#!/bin/sh
# wait-for.sh

set -e

host="$1"
shift
cmd="$@"

until pg_isready -h "$(echo $host | cut -d: -f1)" -p "$(echo $host | cut -d: -f2 || echo 5432)"; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done

>&2 echo "Postgres is up - executing command"
exec $cmd