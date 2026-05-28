#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP_SERVERS="${BOOTSTRAP_SERVERS:-kafka-1:9092,kafka-2:9092,kafka-3:9092}"
KAFKA_TOPICS_BIN="${KAFKA_TOPICS_BIN:-/opt/kafka/bin/kafka-topics.sh}"

if [ ! -x "${KAFKA_TOPICS_BIN}" ]; then
  echo "[init-kafka] kafka topics bin not found: ${KAFKA_TOPICS_BIN}"
  exit 1
fi

echo "[init-kafka] Starting up Kafka cluster: ${BOOTSTRAP_SERVERS}"
for i in $(seq 1 10); do
  if "${KAFKA_TOPICS_BIN}" --bootstrap-server "${BOOTSTRAP_SERVERS}" --list >/dev/null 2>&1; then
    echo "[init-kafka] Kafka ready"
    break
  fi

  if [ "${i}" -eq 10 ]; then
    echo "[init-kafka] Kafka did not ready"
    exit 1
  fi

  sleep 3
done

# order events topic
"${KAFKA_TOPICS_BIN}" --bootstrap-server "${BOOTSTRAP_SERVERS}" --create --if-not-exists \
  --topic "order-events" --partitions 1 --replication-factor 3 --config min.insync.replicas=2

# order events DLQ
"${KAFKA_TOPICS_BIN}" --bootstrap-server "${BOOTSTRAP_SERVERS}" --create --if-not-exists \
  --topic "order-events-dlq" --partitions 1 --replication-factor 3 --config min.insync.replicas=2

echo "[init-kafka] order-events topic"
"${KAFKA_TOPICS_BIN}" --bootstrap-server "${BOOTSTRAP_SERVERS}" --describe --topic "order-events"

echo "[init-kafka] order-events-dlq topic"
"${KAFKA_TOPICS_BIN}" --bootstrap-server "${BOOTSTRAP_SERVERS}" --describe --topic "order-events-dlq"

echo "[init-kafka] Kafka cluster up"
