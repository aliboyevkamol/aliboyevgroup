@echo off
cd /d "%~dp0"
docker run --rm ^
  --network backend_default ^
  -v "%cd%:/src" ^
  -w /src ^
  -e TEST_DATABASE_URL=postgres://kamol:kamol@postgres:5432/kamol_test?sslmode=disable ^
  -e APP_ENV=development ^
  -e JWT_SECRET=dev_jwt_secret_change_me_0123456789abcdef ^
  -e JWT_REFRESH_SECRET=dev_refresh_secret_change_me_0123456789abcdef ^
  -e PAYMENT_WEBHOOK_SECRET=dev_webhook_secret_change_me_0123456789abcdef ^
  -e PAYMENT_PROVIDER=test ^
  golang:1.24.4-alpine3.21 sh -c "apk add --no-cache gcc musl-dev >/dev/null 2>&1; go test ./... 2>&1" > gotest.log 2>&1
echo DONE_EXIT_%ERRORLEVEL% >> gotest.log
