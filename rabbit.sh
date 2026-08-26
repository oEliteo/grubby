#!/bin/bash

start_or_run () {
    docker inspect grubby_rabbitmq > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        echo "Starting grubby RabbitMQ container..."
        docker start grubby_rabbitmq
    else
        echo "grubby RabbitMQ container not found, creating a new one..."
        docker run -d --name grubby_rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:management-alpine
    fi
}

case "$1" in
    start)
        start_or_run
        ;;
    stop)
        echo "Stopping grubby RabbitMQ container..."
        docker stop grubby_rabbitmq
        ;;
    logs)
        echo "Fetching logs for grubby RabbitMQ container..."
        docker logs -f grubby_rabbitmq
        ;;
    *)
        echo "Usage: $0 {start|stop|logs}"
        exit 1
esac
