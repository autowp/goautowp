mkdir -p /var/log/supervisor

waitforit -address tcp://$GOAUTOWP_RABBITMQ_HOST:$GOAUTOWP_RABBITMQ_PORT -timeout 30

./goautowp
