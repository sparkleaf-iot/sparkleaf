#!/bin/bash
exec > /ops/config/output.txt 2>&1
set -e

CONFIGDIR=/ops/config
CONSULCONFIGDIR=/etc/consul.d
NOMADCONFIGDIR=/etc/nomad.d
CONSULTEMPLATECONFIGDIR=/etc/consul-template.d
HOME_DIR=ubuntu
IP_ADDRESS=$(curl -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/network-interfaces/0/ip)
SERVICE_ACCOUNT_KEY=SERVICE_ACCOUNT_KEY_PLACEHOLDER
SERVICE_ACCOUNT=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token) 
CONSUL_BOOTSTRAP_TOKEN=""
NOMAD_BOOTSTRAP_TOKEN=""
NOMAD_USER_TOKEN=""
INSTANCE_NUMBER=INSTANCE_NUMBER_PLACEHOLDER
# Consul
sed -i "s/IP_ADDRESS/$IP_ADDRESS/g" $CONFIGDIR/consul.hcl

#sed -i "s/SERVER_COUNT/$SERVER_COUNT/g" $CONFIGDIR/consul.hcl
#sed -i "s/RETRY_JOIN/$RETRY_JOIN/g" $CONFIGDIR/consul.hcl

# Add hostname to /etc/hosts
echo "127.0.0.1 $(hostname)" | sudo tee --append /etc/hosts

sudo cp $CONFIGDIR/consul.hcl $CONSULCONFIGDIR

sudo systemctl enable consul.service
sudo systemctl start consul.service
export CONSUL_HTTP_ADDR=$IP_ADDRESS:8500
export CONSUL_RPC_ADDR=$IP_ADDRESS:8400

# Move the config for server setup
sudo mv $CONFIGDIR/nomad-server.hcl $NOMADCONFIGDIR/nomad-server.hcl
sed -i "s/CONSUL_TOKEN/nomad_consul_token_secret/g" $NOMADCONFIGDIR/nomad-server.hcl

## Start nomad
sudo systemctl enable nomad.service
sudo systemctl start nomad.service
sudo touch /ops/config/nomad-token.txt
sudo touch /ops/config/nomad-output.txt

sudo touch /ops/config/consul-token.txt
sudo touch /ops/config/consul-output.txt
#CONSUL_BOOTSTRAP_TOKEN=$(echo $OUTPUT | grep -i secretid | awk '{print $4}')
#sudo echo $CONSUL_BOOTSTRAP_TOKEN > /ops/config/consul-token.txt
#sed -i "s/BOOTSTRAP_TOKEN/$CONSUL_BOOTSTRAP_TOKEN/g" $CONSULCONFIGDIR/consul.hcl
# Wait until leader has been elected and bootstrap consul ACLs
for i in {1..20}; do
    # capture stdout and stderr
    set +e
    sleep 5
    OUTPUT=$(consul acl bootstrap 2>&1)
    if [ $? -ne 0 ]; then
        echo "consul acl bootstrap: $OUTPUT" 
        if [[ "$OUTPUT" = *"No cluster leader"* ]]; then
            echo "consul no cluster leader" >> "/ops/config/consul-output.txt"
            continue
        else
            echo "consul already bootstrapped" >> "/ops/config/consul-output.txt"
            exit 0
        fi

    fi
    set -e
    echo "$OUTPUT" >> "/ops/config/consul-output.txt"
    CONSUL_BOOTSTRAP_TOKEN=$(echo "$OUTPUT" | grep -i secretid | awk '{print $2}')
    if [ -n "$CONSUL_BOOTSTRAP_TOKEN" ]; then
        echo "consul bootstrapped" >> "/ops/config/consul-output.txt"
        break
    fi
done
sleep 10
echo "consul loop done" >> "/ops/config/nomad-output.txt"
consul acl policy create -name 'nomad-auto-join' -rules="@$CONFIGDIR/consul-acl-nomad-auto-join.hcl" -token=$CONSUL_BOOTSTRAP_TOKEN

consul acl role create -name "nomad-auto-join" -description "Role with policies necessary for nomad servers and clients to auto-join via Consul." -policy-name "nomad-auto-join" -token=$CONSUL_BOOTSTRAP_TOKEN

consul acl token create -accessor=nomad_consul_token_id -secret=nomad_consul_token_secret -description "Nomad server/client auto-join token" -role-name nomad-auto-join -token=$CONSUL_BOOTSTRAP_TOKEN

echo "nomad loop starting" >> "/ops/config/nomad-output.txt"
# Wait for nomad servers to come up and bootstrap nomad ACL
for i in {1..40}; do
    echo "started loop" >> "/ops/config/nomad-output.txt"
    # capture stdout and stderr
    set +e
    sleep 5
    OUTPUT=$(nomad acl bootstrap 2>&1)
    if [ $? -ne 0 ]; then
        if [[ "$OUTPUT" = *"No cluster leader"* ]]; then
            echo "nomad no cluster leader" >> "/ops/config/nomad-output.txt"
            continue
        else
            echo "nomad already bootstrapped" >> "/ops/config/nomad-output.txt"
            exit 0
        fi
    fi
    set -e
    echo "$OUTPUT" >> "/ops/config/nomad-output.txt"
    NOMAD_BOOTSTRAP_TOKEN=$(echo "$OUTPUT" | grep -i secret | awk -F '=' '{print $2}' | xargs | awk 'NF')   
    if [ -s "$NOMAD_BOOTSTRAP_TOKEN" ]; then
        echo "nomad bootstrapped" >> "/ops/config/nomad-output.txt"
        break
    else
        echo "Problem extracting token" >> "/ops/config/nomad-output.txt"
        break
    fi

done
#NOMAD_BOOTSTRAP_TOKEN=$(cat /ops/config/nomad-output.txt | grep -i secret | awk -F '=' '{print $3}' | xargs | sed 's/.....$//' | awk 'NF' )
#sudo echo $NOMAD_BOOTSTRAP_TOKEN > /ops/config/nomad-token.txt

sleep 10

nomad acl policy apply -token $NOMAD_BOOTSTRAP_TOKEN -description "Policy to allow reading of agents and nodes and listing and submitting jobs in all namespaces." node-read-job-submit $CONFIGDIR/nomad-acl-user.hcl

NOMAD_USER_TOKEN=$(nomad acl token create -token $NOMAD_BOOTSTRAP_TOKEN -name "read-token" -policy node-read-job-submit | grep -i secret | awk -F "=" '{print $2}' | xargs)
echo "acl  done" >> "/ops/config/nomad-output.txt"

# Write user token to kv
consul kv put -token=$CONSUL_BOOTSTRAP_TOKEN "nomad_user_token" $NOMAD_USER_TOKEN
consul kv put -token=$CONSUL_BOOTSTRAP_TOKEN "consul_bt" $CONSUL_BOOTSTRAP_TOKEN

echo "kv done" >> "/ops/config/nomad-output.txt"
# Write service account to kv, used for the csi driver plugin
#DECODED_KEY=$(echo $SERVICE_ACCOUNT | base64 --decode)
#consul kv put -token=$CONSUL_BOOTSTRAP_TOKEN service_account $DECODED_KEY
