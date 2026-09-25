@echo off
REM Generate a self-signed CA and server certificate for local Mosquitto TLS.
REM DEV ONLY -- do not use these certs in production. Keys are gitignored.
REM
REM Windows / cmd.exe equivalent of gen-dev-certs.sh. Requires openssl on PATH
REM (Git for Windows ships one; "openssl version" should work from cmd).
REM Run from anywhere; certs are written next to this script.
setlocal
pushd "%~dp0"

if exist server.crt if exist ca.crt (
  echo certs already present, skipping
  popd & exit /b 0
)

REM CA
openssl req -x509 -newkey rsa:2048 -nodes -keyout ca.key -out ca.crt ^
  -days 3650 -subj "/CN=VleeFruit-Dev-CA"
if errorlevel 1 goto :fail

REM Server key + CSR
openssl req -newkey rsa:2048 -nodes -keyout server.key -out server.csr ^
  -subj "/CN=mosquitto"
if errorlevel 1 goto :fail

REM Sign server cert with SANs so both the in-network name and localhost verify.
REM (openssl on Windows cannot read /dev/stdin, so use a real extension file.)
>san.ext echo subjectAltName=DNS:mosquitto,DNS:localhost,IP:127.0.0.1
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial ^
  -out server.crt -days 3650 -extfile san.ext
if errorlevel 1 goto :fail

del /q server.csr ca.srl san.ext 2>nul
echo generated ca.crt, server.crt, server.key
popd & exit /b 0

:fail
echo cert generation failed 1>&2
del /q server.csr san.ext 2>nul
popd & exit /b 1
