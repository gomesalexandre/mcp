#!/bin/bash

set -e

if [ -z "$SERVER" ] || [ -z "$USER" ] || [ -z "$DEPLOY_PATH" ]; then
    echo "Error: SERVER, USER, and DEPLOY_PATH environment variables must be set"
    exit 1
fi

echo "Deploying to $USER@$SERVER:$DEPLOY_PATH..."

echo "1. Syncing files to server..."
rsync -avz --delete \
    --exclude='.git' \
    --exclude='.devenv' \
    --exclude='.env' \
    --exclude='.github/' \
    --exclude='.claude/' \
    --exclude='*.log' \
    ./ $USER@$SERVER:$DEPLOY_PATH/

echo "2. Building and deploying mcp-server on server..."
ssh $USER@$SERVER << EOF
cd $DEPLOY_PATH
echo "Building mcp-server binary..."
go build -o mcp-server ./cmd/mcp-server/
echo "Stopping mcp-server service before binary replacement..."
sudo systemctl stop mcp-server || true
echo "Installing mcp-server binary to /usr/local/bin/..."
sudo cp mcp-server /usr/local/bin/mcp-server
sudo chmod +x /usr/local/bin/mcp-server
# Verify binary was installed
if [ ! -f "/usr/local/bin/mcp-server" ]; then
    echo "ERROR: mcp-server binary not found in /usr/local/bin/"
    exit 1
fi
echo "Creating application directory..."
sudo mkdir -p /var/lib/mcp
sudo chown $USER:$USER /var/lib/mcp
echo "Binary installation successful:"
ls -la /usr/local/bin/mcp-server
echo "Restarting mcp-server service..."
sudo systemctl restart mcp-server
echo "Checking service status..."
sudo systemctl status mcp-server --no-pager -l
echo "Deployment completed!"
EOF

echo "Deployment finished successfully!"
