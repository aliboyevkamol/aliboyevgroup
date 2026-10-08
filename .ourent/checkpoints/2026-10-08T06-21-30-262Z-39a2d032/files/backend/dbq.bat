@echo off
docker exec backend-postgres-1 psql -U kamol -d kamol -A -F"|" -c "%~1" | more +1
