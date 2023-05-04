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
sed -i "s/IP_ADDRESS/$IP_ADDRESS/g" $CONFIGDIR/consul.hcl
#sed -i "s/SERVER_COUNT/$SERVER_COUNT/g" $CONFIGDIR/consul.hcl
#sed -i "s/RETRY_JOIN/$RETRY_JOIN/g" $CONFIGDIR/consul.hcl
sudo cp $CONFIGDIR/consul.hcl $CONSULCONFIGDIR

sudo systemctl enable consul.service
sudo systemctl start consul.service
sleep 10
set +e
OUTPUT=$(consul acl bootstrap 2>&1)
sudo touch /ops/config/token.txt
BOOTSTRAP_TOKEN=$(echo $OUTPUT | grep -i secretid | awk '{print $4}')
sudo echo $BOOTSTRAP_TOKEN > /ops/config/token.txt

consul acl policy create -name 'nomad-auto-join' -rules="@$CONFIGDIR/consul-acl-nomad-auto-join.hcl" -token=$BOOTSTRAP_TOKEN

consul acl role create -name "nomad-auto-join" -description "Role with policies necessary for nomad servers and clients to auto-join via Consul." -policy-name "nomad-auto-join" -token=$BOOTSTRAP_TOKEN

consul acl token create -accessor=nomad_consul_token_id -secret=nomad_consul_token_secret -description "Nomad server/client auto-join token" -role-name nomad-auto-join -token=$BOOTSTRAP_TOKEN

#sed -i "s/BOOTSTRAP_TOKEN/$BOOTSTRAP_TOKEN/g" $CONFIGDIR/consul.hcl
consul reload

#consul acl policy create -name 'consul-user' -rules="@$CONFIGDIR/consul-acl-user.hcl" -token-file=$BOOTSTRAP_TOKEN
#consul acl role create -name "consul-user" -description "Role to login to consul" -policy-name "nomad-auto-join" -token-file=$BOOTSTRAP_TOKEN

# Move the config for server setup
sudo mv $CONFIGDIR/nomad-server.hcl $NOMADCONFIGDIR/nomad-server.hcl

## Start
sudo systemctl enable nomad
sudo systemctl start nomad

