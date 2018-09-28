@echo off
setlocal

:: Create self-signed certificate.
call:mkdirp %~dp0certs
call openssl req ^
  -newkey rsa:2048 ^
  -nodes ^
  -x509 ^
  -days 365 ^
  -subj "/C=US/ST=Denial/L=Springfield/O=Dis/CN=127.0.0.1" ^
  -keyout %~dp0certs\registry_auth.key ^
  -out %~dp0certs\registry_auth.crt
if %ErrorLevel% neq 0 (
  call:print "Failed to generate self-signed certificate."
  exit /b %ErrorLevel%
)

:: Generate random credentials.
call:mkdirp %~dp0auth
call docker run ^
  --rm ^
  --entrypoint htpasswd ^
  registry:2 ^
  -Bbn testuser testpwd ^
  > %~dp0auth\htpasswd
if %ErrorLevel% neq 0 (
  call:print "Failed to generate random credentials."
  exit /b %ErrorLevel%
)

:: Start an ephemeral Docker registry in a container.
::  --rm ^
@echo on
call docker run ^
  -d ^
  --name private_registry ^
  -p 15000:5000 ^
  -v %~dp0auth:/auth ^
  -e "REGISTRY_AUTH=htpasswd" ^
  -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" ^
  -e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" ^
  -v %~dp0certs:/certs ^
  -e "REGISTRY_HTTP_TLS_CERTIFICATE=/certs/registry_auth.crt" ^
  -e "REGISTRY_HTTP_TLS_KEY=/certs/registry_auth.key" ^
  registry:2
if %ErrorLevel% neq 0 (
  call:print "Failed to create ephemeral Docker registry."
  exit /b %ErrorLevel%
)

:: Wait until the container is responsive (*crosses fingers*).
timeout /t 5

:: Point our Docker Daemon to this ephemeral registry.
call docker login -u testuser -p testpwd 127.0.0.1:15000
if %ErrorLevel% neq 0 (
  call:print "Failed to log in to ephemeral Docker registry."
  exit /b %ErrorLevel%
)

:: Build a few private images.
for /L %%i in (1,1,3) do (
  call docker build ^
    -t tftest-service ^
    %~dp0 ^
    -f %~dp0Dockerfile_v%%i
  call docker tag ^
    tftest-service ^
    127.0.0.1:15000/tftest-service:v%%i
  call docker push ^
    127.0.0.1:15000/tftest-service:v%%i
)

exit /b %ErrorLevel%


:print
  echo %~1
  exit /b 0


:mkdirp
  if not exist %~1\nul (
    mkdir %~1
  )
  exit /b %ErrorLevel%
