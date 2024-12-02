#!/bin/bash

echo "$SSH_KEY" > key.pem
chmod 600 key.pem

scp -o StrictHostKeyChecking=no -i key.pem cmd/node/unchain cmd/node/unchain.service $SSH_USER@$SSH_HOST:~
ssh -o StrictHostKeyChecking=no -i key.pem $SSH_USER@$SSH_HOST << EOF
  cd ~ && pwd
  sudo rm -rf /app && sudo mkdir /app
  sudo mv unchain /app/unchain
  sudo chmod +x /app/unchain
  echo "$CONFIG_TOML" | sudo tee /app/config.toml > /dev/null
  sudo mv unchain.service /etc/systemd/system/unchain.service
  sudo rm -rf /etc/systemd/system/emissary.service
  sudo systemctl daemon-reload
  sudo systemctl stop emissary.service || true
  sudo systemctl stop unchain.service || true
  sudo systemctl start unchain.service
  sudo systemctl status unchain.service
EOF