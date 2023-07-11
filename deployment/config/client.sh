#!/bin/bash
exec > /ops/config/output.txt 2>&1

set -e

CONFIGDIR=/ops/config
CONSULCONFIGDIR=/etc/consul.d
NOMADCONFIGDIR=/etc/nomad.d
CONSULTEMPLATECONFIGDIR=/etc/consul-template.d
HOME_DIR=ubuntu
BOOTSTRAP_TOKEN=nomad_consul_token_secret
CLOUD_ENV=gce

# Wait for network
sleep 15


# Add hostname to /etc/hosts
echo "127.0.0.1 $(hostname)" | sudo tee --append /etc/hosts

# Fetch ip of current vm instance
IP_ADDRESS=$(curl -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
export NOMAD_ADDR=http://$IP_ADDRESS:4646

# Consul
export CONSUL_HTTP_ADDR=$IP_ADDRESS:8500
export CONSUL_RPC_ADDR=$IP_ADDRESS:8400
sudo systemctl enable consul.service
sudo systemctl start consul.service
sleep 30
set +e
OUTPUT=$(consul acl bootstrap 2>&1)
sudo touch /ops/config/consul-token.txt
CONSUL_BOOTSTRAP_TOKEN=$(echo $OUTPUT | grep -i secretid | awk '{print $4}')
sudo echo $CONSUL_BOOTSTRAP_TOKEN > /ops/config/consul-token.txt
#sed -i "s/BOOTSTRAP_TOKEN/$CONSUL_BOOTSTRAP_TOKEN/g" $CONFIGDIR/consul-client.hcl

sed -i "s/IP_ADDRESS/$IP_ADDRESS/g" $CONFIGDIR/consul-client.hcl
#sed -i "s/RETRY_JOIN/$RETRY_JOIN/g" $CONFIGDIR/consul-client.hcl
sudo cp $CONFIGDIR/consul-client.hcl $CONSULCONFIGDIR



# Move the config for client setup
sed -i "s/CONSUL_TOKEN/nomad_consul_token_secret/g" $CONFIGDIR/nomad-client.hcl
sudo mv $CONFIGDIR/nomad-client.hcl $NOMADCONFIGDIR/nomad-client.hcl
## Start nomad
sudo systemctl enable nomad.service
sudo systemctl start nomad.service


