#!/bin/bash

set -e

CONFIGDIR=/ops/config
CONSULCONFIGDIR=/etc/consul.d
NOMADCONFIGDIR=/etc/nomad.d
CONSULTEMPLATECONFIGDIR=/etc/consul-template.d
HOME_DIR=ubuntu
BOOTSTRAP_TOKEN=BOOTSTRAP_TOKEN_PLACEHOLDER
CLOUD_ENV=gce

# Wait for network
sleep 15
# Replace token in nomad client config
sed -i "s/CONSUL_TOKEN/nomad_consul_token_secret/g" $CONFIGDIR/nomad-client.hcl

# Fetch ip of current vm instance
IP_ADDRESS=$(curl -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
export NOMAD_ADDR=http://$IP_ADDRESS:4646

# Consul
sed -i "s/BOOTSTRAP_TOKEN/$BOOTSTRAP_TOKEN/g" $CONFIGDIR/consul-client.hcl
sed -i "s/IP_ADDRESS/$IP_ADDRESS/g" $CONFIGDIR/consul-client.hcl
#sed -i "s/RETRY_JOIN/$RETRY_JOIN/g" $CONFIGDIR/consul-client.hcl
sudo cp $CONFIGDIR/consul-client.hcl $CONSULCONFIGDIR

sudo systemctl enable consul.service
sudo systemctl start consul.service
sleep 10

# Move the config for client setup
sudo mv $CONFIGDIR/nomad-client.hcl $NOMADCONFIGDIR/nomad-client.hcl
## Start
sudo systemctl enable nomad.service
sudo systemctl start nomad.service


