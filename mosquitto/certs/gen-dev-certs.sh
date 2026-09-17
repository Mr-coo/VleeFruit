#!/bin/sh
# Generate a self-signed CA and server certificate for local Mosquitto TLS.
# DEV ONLY — do not use these certs in production. Keys are gitignored.
#
# Run from anywhere; certs are written next to this script.
set -eu
cd "$(dirname "$0")"

if [ -f server.crt ] && [ -f ca.crt ]; then
  echo "certs already present, skipping"
  exit 0
fi

# CA
openssl req -x509 -newkey rsa:2048 -nodes -keyout ca.key -out ca.crt \
  -days 3650 -subj "/CN=VleeFruit-Dev-CA"

# Server key + CSR
openssl req -newkey rsa:2048 -nodes -keyout server.key -out server.csr \
  -subj "/CN=mosquitto"

# Sign server cert with SANs so both the in-network name and localhost verify.
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 3650 \
  -extfile /dev/stdin <<'EXT'
subjectAltName=DNS:mosquitto,DNS:localhost,IP:127.0.0.1
EXT

rm -f server.csr ca.srl
# Mosquitto runs as a non-root user; make the key readable (dev certs only).
chmod 644 ca.crt ca.key server.crt server.key
echo "generated ca.crt, server.crt, server.key"
