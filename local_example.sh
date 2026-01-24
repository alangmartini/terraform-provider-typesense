#!/bin/bash

### Setup Local Typesense for Acceptance Tests ########################################

# Configuration
export TYPESENSE_API_KEY=test-api-key-for-acceptance-tests
export PORT=8108
export CONTAINER_NAME=typesense-test

# For acceptance tests, the provider expects these environment variables
export TYPESENSE_HOST=localhost
export TYPESENSE_PORT=$PORT
export TYPESENSE_PROTOCOL=http

echo "Setting up local Typesense instance for acceptance tests..."

# Clean up any existing container and data
docker stop $CONTAINER_NAME 2>/dev/null
docker rm $CONTAINER_NAME 2>/dev/null
rm -rf "$(pwd)"/typesense-test-data
mkdir "$(pwd)"/typesense-test-data

# Start Typesense container
echo "Starting Typesense container..."
docker run -d -p $PORT:$PORT --name $CONTAINER_NAME \
            -v"$(pwd)"/typesense-test-data:/data \
            typesense/typesense:29.0.rc30 \
            --data-dir /data \
            --api-key=$TYPESENSE_API_KEY \
            --enable-cors

# Wait for Typesense to be ready
echo "Waiting for Typesense to be ready..."
until curl -s -o /dev/null -w "%{http_code}" "http://localhost:$PORT/health" -H "X-TYPESENSE-API-KEY: ${TYPESENSE_API_KEY}" | grep -q "200"; do
  echo "  Still waiting..."
  sleep 2
done

echo ""
echo "✓ Typesense is ready!"
echo ""
echo "Environment variables set:"
echo "  TYPESENSE_HOST=$TYPESENSE_HOST"
echo "  TYPESENSE_PORT=$TYPESENSE_PORT"
echo "  TYPESENSE_PROTOCOL=$TYPESENSE_PROTOCOL"
echo "  TYPESENSE_API_KEY=$TYPESENSE_API_KEY"
echo ""
echo "To run acceptance tests, use:"
echo "  TF_ACC=1 go test ./... -v"
echo ""
echo "To stop the test cluster, run:"
echo "  docker stop $CONTAINER_NAME && docker rm $CONTAINER_NAME"
echo ""

### Documentation ######################################################################################
# Visit the API reference section: https://typesense.org/docs/28.0/api/collections.html
# Click on the "Shell" tab under each API resource's docs, to get shell commands for other API endpoints