FROM ubuntu

LABEL app_name="autowp.goautowp"
LABEL maintainer="dmitry@pereslegin.ru"

ENV GOAUTOWP_INPUT_HOST="localhost" \
    GOAUTOWP_INPUT_PORT="5672" \
    GOAUTOWP_INPUT_QUEUE="input" \
    GOAUTOWP_MYSQL_DSN="root:password@tcp(localhost:3306)/autowp?charset=utf8mb4&parseTime=true&loc=UTC" \
    GOAUTOWP_MIGRATIONS_DSN="mysql://root:password@tcp(localhost)/autowp?charset=utf8mb4&parseTime=true&loc=UTC" \
    GOAUTOWP_SENTRY_DSN="" \
    GOAUTOWP_SENTRY_ENVIRONMENT="production"

EXPOSE 80

VOLUME /var/log
  
WORKDIR /app

ENTRYPOINT ["sh", "/entrypoint.sh"]

COPY entrypoint.sh /entrypoint.sh

RUN DEBIAN_FRONTEND=noninteractive apt-get update -qq -y && \
    DEBIAN_FRONTEND=noninteractive apt-get dist-upgrade -qq -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -qq -y \
        ca-certificates \
        wget && \
    DEBIAN_FRONTEND=noninteractive apt-get autoremove -y && \
    DEBIAN_FRONTEND=noninteractive apt-get autoclean -y && \
    \
    wget -q -O /usr/local/bin/waitforit https://github.com/maxcnunes/waitforit/releases/download/v2.4.1/waitforit-linux_amd64 && \
    chmod +x /usr/local/bin/waitforit
 
COPY goautowp /app/goautowp
COPY migrations /app/migrations
 
