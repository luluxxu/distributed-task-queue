#!/bin/bash
# Helper script to update client code with API endpoint

if [ -z "$1" ]; then
  echo "Usage: $0 <api-endpoint>"
  echo "Example: $0 http://54.123.45.67:8080"
  exit 1
fi

API_ENDPOINT=$1
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "Updating client code to use: $API_ENDPOINT"

# Update exp1
sed -i.bak "s|http://localhost:8080|$API_ENDPOINT|g" "$BASE_DIR/src/client/exp1/exp1_loadtest.go"
echo "✓ Updated exp1_loadtest.go"

# Update exp2
sed -i.bak "s|baseURL.*=.*\"http://localhost:8080\"|baseURL = \"$API_ENDPOINT\"|g" "$BASE_DIR/src/client/exp2/exp2_loadtest.go"
echo "✓ Updated exp2_loadtest.go"

# Update exp3
sed -i.bak "s|baseURL := \"http://localhost:8080\"|baseURL := \"$API_ENDPOINT\"|g" "$BASE_DIR/src/client/exp3/exp3_loadtest.go"
echo "✓ Updated exp3_loadtest.go"

echo ""
echo "Backup files created with .bak extension"
echo "To restore: find src/client -name '*.bak' -exec sh -c 'mv \"\$1\" \"\${1%.bak}\"' _ {} \;"

