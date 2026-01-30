#!/bin/sh
set -eu

tries=0
max_tries=30

until kafka-topics --bootstrap-server kafka:29092 --list >/dev/null 2>&1; do
  tries=$((tries + 1))
  if [ "$tries" -ge "$max_tries" ]; then
    echo "kafka not ready after ${max_tries} tries" >&2
    exit 1
  fi
  sleep 2
 done

kafka-topics --bootstrap-server kafka:29092 --create --if-not-exists --topic submissions --partitions 1 --replication-factor 1
kafka-topics --bootstrap-server kafka:29092 --create --if-not-exists --topic results --partitions 1 --replication-factor 1