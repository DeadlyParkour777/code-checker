#!/bin/sh
set -eu

kafka-topics --bootstrap-server kafka:29092 --create --if-not-exists --topic submissions --partitions 1 --replication-factor 1
kafka-topics --bootstrap-server kafka:29092 --create --if-not-exists --topic results --partitions 1 --replication-factor 1
