#!/bin/bash


start_or_run () {
    docker inspect grubbytest_rabbitmq > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        echo "Starting grubbytest RabbitMQ container..."
        docker start grubbytest_rabbitmq
    else
        echo "grubbytest RabbitMQ container not found, creating a new one..."
        docker run -d --name grubbytest_rabbitmq \
           -p 5672:5672 -p 15672:15672 \
           -e RABBITMQ_DEFAULT_USER="test" \
           -e RABBITMQ_DEFAULT_PASS="test" \
           rabbitmq:management-alpine
    fi
}

case "$1" in
    start)
        start_or_run
        ;;
    stop)
        echo "Stopping grubbytest RabbitMQ container..."
        docker stop grubbytest_rabbitmq
        ;;
    logs)
        echo "Fetching logs for grubbytest RabbitMQ container..."
        docker logs -f grubbytest_rabbitmq
        ;;
    *)
        echo "Usage: $0 {start|stop|logs}"
        exit 1
esac
