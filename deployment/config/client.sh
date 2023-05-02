#!/bin/bash

set -e

CONFIGDIR=/ops/config
CONSULCONFIGDIR=/etc/consul.d
NOMADCONFIGDIR=/etc/nomad.d
CONSULTEMPLATECONFIGDIR=/etc/consul-template.d
HOME_DIR=ubuntu
BOOTSTRAP_TOKEN=BOOTSTRAP_TOKEN_PLACEHOLDER
# Wait for network
sleep 15

# Fetch ip of current vm instance
IP_ADDRESS=$(curl -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)


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


