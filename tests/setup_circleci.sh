#!/bin/bash

set -euo pipefail

setup_circleci(){
  if [ "$CIRCLECI" = "true" ];then
     sudo apt-get update
     sudo apt-get install -y make xfsprogs lvm2 go-md2man thin-provisioning-tools

     # Remove default golang (1.10.3) and install a custom version (1.14.3) of golang.
     # This is required for supporting go mod, and to be able to compile docker-lvm-plugin.
     sudo rm -rf /usr/local/go

     # Install golang 1.14.3
     curl -L -o go1.14.3.linux-amd64.tar.gz https://dl.google.com/go/go1.14.3.linux-amd64.tar.gz
     sudo tar -C /usr/local -xzf go1.14.3.linux-amd64.tar.gz
     sudo chmod +x /usr/local/go
     rm -f go1.14.3.linux-amd64.tar.gz
     make
     sudo make install
     sudo systemctl start docker-lvm-plugin
     # Sleep for 5 seconds, to allow docker-lvm-plugin to start completely.
     sleep 5s
     echo "INFO: Circleci setup successful."
  fi
}

setup_circleci
