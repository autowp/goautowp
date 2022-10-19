#!/bin/sh

set -e

TEMPLATES="goautowp-autoban.yaml goautowp-listen-df-amqp.yaml goautowp-listen-monitoring-amqp.yaml goautowp-scheduler-daily.yaml goautowp-scheduler-hourly.yaml goautowp-scheduler-midnight.yaml goautowp-serve-private.yaml goautowp-serve-public.yaml"

for TEMPLATE in ${TEMPLATES}; do
  sed -i -E "s|(image:[[:space:]]+registry\.pereslegin\.ru/autowp/goautowp:).+|image: ${CI_REGISTRY_IMAGE}:${CI_COMMIT_TAG}|" templates/$TEMPLATE
  git add templates/$TEMPLATE
done

git commit -m "feat: Update go-backend to $CI_COMMIT_TAG"
