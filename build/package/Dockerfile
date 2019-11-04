FROM ubuntu

LABEL app_name="autowp.goautowp"
LABEL maintainer="dmitry@pereslegin.ru"

ENV GOAUTOWP_INPUT_HOST="localhost" \
    GOAUTOWP_INPUT_PORT="5672" \
    GOAUTOWP_INPUT_QUEUE="input" \
    GOAUTOWP_MYSQL_HOST="localhost" \
    GOAUTOWP_MYSQL_PORT="3306" \
    GOAUTOWP_MYSQL_USERNAME="root" \
    GOAUTOWP_MYSQL_PASSWORD="root" \
    GOAUTOWP_MYSQL_DBNAME="autowp" \
    GOAUTOWP_SENTRY_DSN="" \
    GOAUTOWP_SENTRY_ENVIRONMENT="production"

EXPOSE 80

VOLUME /var/log
  
WORKDIR /app

RUN DEBIAN_FRONTEND=noninteractive apt-get update -qq -y && \
    DEBIAN_FRONTEND=noninteractive apt-get dist-upgrade -qq -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -qq -y \
        ca-certificates \
        supervisor \
        wget && \
    DEBIAN_FRONTEND=noninteractive apt-get autoremove -y && \
    DEBIAN_FRONTEND=noninteractive apt-get autoclean -y && \
    \
    wget -q -O /usr/local/bin/waitforit https://github.com/maxcnunes/waitforit/releases/download/v2.4.0/waitforit-linux_amd64 && \
    chmod +x /usr/local/bin/waitforit
 
COPY etc /etc
 
COPY goautowp /app/goautowp
COPY migrations /app/migrations
 
COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["sh", "/entrypoint.sh"]
