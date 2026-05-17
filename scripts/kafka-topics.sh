#!/usr/bin/env sh
set -eu

BROKER_CONTAINER="${BROKER_CONTAINER:-golang-microservices-kafka-1}"

TOPICS="market.price.updated market.candle.created strategy.signal.generated risk.order.approved order.created execution.submitted portfolio.updated notification.send.requested dead-letter"

for topic in $TOPICS; do
  docker compose exec -T kafka kafka-topics.sh --create --if-not-exists --topic "$topic" --partitions 3 --replication-factor 1 --bootstrap-server localhost:9092
 done
