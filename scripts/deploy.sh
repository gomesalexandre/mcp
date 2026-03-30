#!/bin/bash

set -e

if [ -z "$SERVER" ] || [ -z "$USER" ] || [ -z "$DEPLOY_PATH" ]; then
    echo "Error: SERVER, USER, and DEPLOY_PATH environment variables must be set"
    exit 1
fi

# Guard against dangerous DEPLOY_PATH values
case "$DEPLOY_PATH" in
    /|/usr|/etc|/var|/home|/root)
        echo "Error: DEPLOY_PATH '$DEPLOY_PATH' is too broad for --delete"
        exit 1
        ;;
esac

SSH_OPTS="-o StrictHostKeyChecking=no"

echo "Deploying to $USER@$SERVER:$DEPLOY_PATH..."

echo "1. Syncing files to server..."
rsync -avz --delete -e "ssh $SSH_OPTS" \
    --exclude='.git' \
    --exclude='.devenv' \
    --exclude='.env' \
    --exclude='.github/' \
    --exclude='.claude/' \
    --exclude='*.log' \
    ./ $USER@$SERVER:$DEPLOY_PATH/

echo "2. Building and deploying mcp-server on server..."
ssh $SSH_OPTS $USER@$SERVER bash -s <<DEPLOY_EOF
export PATH=\$PATH:/usr/local/go/bin
cd $DEPLOY_PATH
echo "Building mcp-server binary..."
go build -o mcp-server ./cmd/mcp-server/
echo "Stopping mcp-server service..."
sudo systemctl stop mcp-server || true
echo "Installing binary to /usr/local/bin/..."
sudo cp mcp-server /usr/local/bin/mcp-server
sudo chmod +x /usr/local/bin/mcp-server
if [ ! -f "/usr/local/bin/mcp-server" ]; then
    echo "ERROR: mcp-server binary not found"
    exit 1
fi
echo "Restarting mcp-server service..."
sudo systemctl restart mcp-server
echo "Checking service status..."
sudo systemctl status mcp-server --no-pager -l || true
echo "Deployment completed!"
DEPLOY_EOF

echo "Deployment finished successfully!"
