#!/bin/bash

set -e

CONFIGDIR=/ops/config
CONSULCONFIGDIR=/etc/consul.d
NOMADCONFIGDIR=/etc/nomad.d
CONSULTEMPLATECONFIGDIR=/etc/consul-template.d
HOME_DIR=ubuntu
IP_ADDRESS=$(curl -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
SERVICE_ACCOUNT=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token) 
CONSUL_BOOTSTRAP_TOKEN=BOOTSTRAP_TOKEN_PLACEHOLDER
NOMAD_BOOTSTRAP_TOKEN=BOOTSTRAP_TOKEN_PLACEHOLDER
NOMAD_USER_TOKEN="/tmp/nomad_user_token"
# Consul
sed -i "s/IP_ADDRESS/$IP_ADDRESS/g" $CONFIGDIR/consul.hcl
#sed -i "s/SERVER_COUNT/$SERVER_COUNT/g" $CONFIGDIR/consul.hcl
#sed -i "s/RETRY_JOIN/$RETRY_JOIN/g" $CONFIGDIR/consul.hcl

# Add hostname to /etc/hosts
echo "127.0.0.1 $(hostname)" | sudo tee --append /etc/hosts

sudo cp $CONFIGDIR/consul.hcl $CONSULCONFIGDIR

sudo systemctl enable consul.service
sudo systemctl start consul.service
sleep 10
set +e
OUTPUT=$(consul acl bootstrap 2>&1)
sudo touch /ops/config/consul-token.txt
CONSUL_BOOTSTRAP_TOKEN=$(echo $OUTPUT | grep -i secretid | awk '{print $4}')
sudo echo $CONSUL_BOOTSTRAP_TOKEN > /ops/config/consul-token.txt

consul acl policy create -name 'nomad-auto-join' -rules="@$CONFIGDIR/consul-acl-nomad-auto-join.hcl" -token=$CONSUL_BOOTSTRAP_TOKEN

consul acl role create -name "nomad-auto-join" -description "Role with policies necessary for nomad servers and clients to auto-join via Consul." -policy-name "nomad-auto-join" -token=$CONSUL_BOOTSTRAP_TOKEN

consul acl token create -accessor=nomad_consul_token_id -secret=nomad_consul_token_secret -description "Nomad server/client auto-join token" -role-name nomad-auto-join -token=$CONSUL_BOOTSTRAP_TOKEN

consul reload

# Move the config for server setup
sudo mv $CONFIGDIR/nomad-server.hcl $NOMADCONFIGDIR/nomad-server.hcl
sed -i "s/CONSUL_TOKEN/nomad_consul_token_secret/g" $NOMADCONFIGDIR/nomad-server.hcl

## Start
sudo systemctl enable nomad
sudo systemctl start nomad
sleep 10
OUTPUT=$(nomad acl bootstrap 2>&1)
sudo touch /ops/config/nomad-token.txt
sudo touch /ops/config/nomad-output.txt

sudo echo $OUTPUT > /ops/config/nomad-output.txt
NOMAD_BOOTSTRAP_TOKEN=$(cat /ops/config/nomad-output.txt | grep -i secret | awk -F '=' '{print $3}' | xargs | sed 's/.....$//' | awk 'NF' )
sudo echo $NOMAD_BOOTSTRAP_TOKEN > /ops/config/nomad-token.txt


nomad acl policy apply -token=$NOMAD_BOOTSTRAP_TOKEN -description "Policy to allow reading of agents and nodes and listing and submitting jobs in all namespaces." node-read-job-submit $CONFIGDIR/nomad-acl-user.hcl

nomad acl token create -token=$NOMAD_BOOTSTRAP_TOKEN -name "read-token" -policy node-read-job-submit | grep -i secret | awk -F "=" '{print $2}' | xargs > $NOMAD_USER_TOKEN

# Write user token to kv
consul kv put -token=$CONSUL_BOOTSTRAP_TOKEN nomad_user_token $(cat $NOMAD_USER_TOKEN)
# Write service account to kv, used for the csi driver plugin
consul kv put -token=$CONSUL_BOOTSTRAP_TOKEN GOOGLE_APPLICATION_CREDENTIALS $SERVICE_ACCOUNT

