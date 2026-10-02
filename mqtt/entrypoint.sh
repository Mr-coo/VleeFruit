#!/bin/sh
# Generates the Mosquitto password and ACL files from the environment, then
# starts the broker. Fails closed: missing credentials stop the container
# instead of starting an open broker.
set -eu

: "${MQTT_USERNAME:?MQTT_USERNAME is required (backend broker user)}"
: "${MQTT_PASSWORD:?MQTT_PASSWORD is required (backend broker password)}"
: "${MQTT_DEVICE_USERNAME:?MQTT_DEVICE_USERNAME is required (device broker user)}"
: "${MQTT_DEVICE_PASSWORD:?MQTT_DEVICE_PASSWORD is required (device broker password)}"

dir=/mosquitto/auth
mkdir -p "$dir"
rm -f "$dir/passwd" "$dir/acl"
touch "$dir/passwd"
chmod 0700 "$dir/passwd"
mosquitto_passwd -b "$dir/passwd" "$MQTT_USERNAME" "$MQTT_PASSWORD"
mosquitto_passwd -b "$dir/passwd" "$MQTT_DEVICE_USERNAME" "$MQTT_DEVICE_PASSWORD"

# Backend: full access to the app topics, plus $SYS for the healthcheck.
# Device: may only send requests and read responses.
cat > "$dir/acl" <<ACL
user $MQTT_USERNAME
topic readwrite vleefruit/#
topic read \$SYS/#

user $MQTT_DEVICE_USERNAME
topic write vleefruit/request
topic write vleefruit/image/request
topic read vleefruit/response
topic read vleefruit/image/response
ACL
chmod 0700 "$dir/acl"
chown -R mosquitto:mosquitto "$dir"

exec /docker-entrypoint.sh /usr/sbin/mosquitto -c /mosquitto/config/mosquitto.conf
