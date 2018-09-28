@echo off
setlocal

:: As of `go-dockerclient` v1.2.0, the default endpoint to the Docker daemon
:: is a UNIX socket.  We need to force it to use the Windows named pipe when
:: running against Docker for Windows.
set DOCKER_HOST=npipe:////.//pipe//docker_engine

:: Note: quoting these values breaks the tests!
set DOCKER_REGISTRY_ADDRESS=127.0.0.1:15000
set DOCKER_REGISTRY_USER=testuser
set DOCKER_REGISTRY_PASS=testpwd
set DOCKER_PRIVATE_IMAGE=127.0.0.1:15000/tftest-service:v1
set TF_ACC=1

call:setup
if %ErrorLevel% neq 0 (
  call:print "Failed to set up acceptance test fixtures."
  exit /b %ErrorLevel%
)

call:run
if %ErrorLevel% neq 0 (
  call:print "Acceptance tests failed."
  set outcome=1
) else (
  set outcome=0
)

call:cleanup
if %ErrorLevel% neq 0 (
  call:print "Failed to clean up acceptance test fixtures."
  exit /b %ErrorLevel%
)

exit /b %outcome%


:print
  if "%~1" == "" (
    echo.
  ) else (
    echo %~1
  ) 
  exit /b 0


:log
  call:print ""
  call:print "##################################"
  call:print "-------- %~1"
  call:print "##################################"
  exit /b 0


:setup
  call:log "setup"
  call %~dp0testing\setup_private_registry.bat
  exit /b %ErrorLevel%


:run
  call:log "run"
  call go test ./docker -v -timeout 120m
  exit /b %ErrorLevel%


:cleanup
  call:log "cleanup"
  call:print "### unsetted env ###"
  for /F %%p in ('docker container ls -f "name=private_registry" -q') do (
    call docker stop %%p
    call docker rm -f -v %%p
  )
  call:print "### stopped private registry ###"
  rmdir /q /s %~dp0testing\auth
  rmdir /q /s %~dp0testing\certs
  call:print "### removed auth and certs ###"
  for %%r in ("container" "volume") do (
    call docker %%r ls -f "name=tftest-" -q
    for /F %%i in ('docker %%r ls -f "name=tf-test" -q') do (
      echo Deleting %%r %%i
      call docker %%r rm -f -v %%i
    )
    for /F %%i in ('docker %%r ls -f "name=tftest-" -q') do (
      echo Deleting %%r %%i
      call docker %%r rm -f -v %%i
    )
    call:print "### removed %%r ###"
  )
  for %%r in ("config" "secret" "network") do (
    call docker %%r ls -f "name=tftest-" -q
    for /F %%i in ('docker %%r ls -f "name=tftest-" -q') do (
      echo Deleting %%r %%i
      call docker %%r rm %%i
    )
    call:print "### removed %%r ###"
  )
  for /F %%i in ('docker images -aq 127.0.0.1:5000/tftest-service') do (
    echo Deleting imag %%i
    docker rmi -f %%i
  )
  call:print "### removed service images ###"
  exit /b %ErrorLevel%
