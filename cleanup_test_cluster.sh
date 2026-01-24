#!/bin/bash

### Cleanup Test Typesense Cluster ########################################

CONTAINER_NAME=typesense-test

echo "Stopping and removing Typesense test container..."
docker stop $CONTAINER_NAME 2>/dev/null
docker rm $CONTAINER_NAME 2>/dev/null

echo "Removing test data directory..."
rm -rf "$(pwd)"/typesense-test-data

echo "✓ Cleanup complete!"
