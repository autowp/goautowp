#!/bin/sh

set -e

templates=(
  "goautowp-autoban.yaml"
  "goautowp-listen-df-amqp.yaml"
  "goautowp-listen-monitoring-amqp.yaml"
  "goautowp-scheduler-daily.yaml"
  "goautowp-scheduler-hourly.yaml"
  "goautowp-scheduler-midnight.yaml"
  "goautowp-serve-private.yaml"
  "goautowp-serve-public.yaml"

)

for template in ${templates[@]}; do
  sed -i -E "s/(image:[[:space:]]+registry\.pereslegin\.ru\/autowp\/goautowp:).+/image: $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG/" templates/$template
  git add templates/$template
done

git commit -m "feat: Update go-backend to $CI_COMMIT_TAG"
