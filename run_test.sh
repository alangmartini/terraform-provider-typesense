export CONTAINER_NAME=typesense-test
export TYPESENSE_PORT=8108
export TYPESENSE_API_KEY=test-api-key-for-acceptance-tests
export PORT=8108

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


export TF_ACC=1
export TYPESENSE_HOST="localhost"
export TYPESENSE_API_KEY=$TYPESENSE_API_KEY
export TYPESENSE_PORT=$PORT
export TYPESENSE_PROTOCOL="http"                                                                                                 
                                                                                                                                        
make test