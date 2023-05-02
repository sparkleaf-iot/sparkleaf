#!/bin/bash

set -e

CONFIGDIR=/ops/config
CONSULCONFIGDIR=/etc/consul.d
NOMADCONFIGDIR=/etc/nomad.d
CONSULTEMPLATECONFIGDIR=/etc/consul-template.d
HOME_DIR=ubuntu
IP_ADDRESS=$(curl -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
BOOTSTRAP_TOKEN=BOOTSTRAP_TOKEN_PLACEHOLDER


# Consul
sed -i "s/BOOTSTRAP_TOKEN/$BOOTSTRAP_TOKEN/g" $CONFIGDIR/consul.hcl
sed -i "s/IP_ADDRESS/$IP_ADDRESS/g" $CONFIGDIR/consul.hcl
#sed -i "s/SERVER_COUNT/$SERVER_COUNT/g" $CONFIGDIR/consul.hcl
#sed -i "s/RETRY_JOIN/$RETRY_JOIN/g" $CONFIGDIR/consul.hcl
sudo cp $CONFIGDIR/consul.hcl $CONSULCONFIGDIR

sudo systemctl enable consul.service
sudo systemctl start consul.service
sleep 10

# Move the config for server setup
sudo mv $CONFIGDIR/nomad-server.hcl $NOMADCONFIGDIR/nomad-server.hcl

## Start
sudo systemctl enable nomad
sudo systemctl start nomad

