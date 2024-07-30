#!/bin/bash
##
## Copyright (c) Microsoft Corporation.
## Licensed under the MIT license.
## SPDX-License-Identifier: MIT
##

inputs_file=$1

output_file="${inputs_file%.*}-output.${inputs_file##*.}"

# Generate timestamp
timestamp=$(date +"%Y%m%d_%H%M%S")

# Append timestamp to log file name
logFile="/var/log/staging_$timestamp.log"

# Function to handle errors
error_handler() {
    local lineno=$1
    local msg=$2
    echo "Error occurred in script at line: $lineno" | tee -a $logFile
    echo "Error message: $msg" | tee -a $logFile
    echo "{\"status\":500}" | jq -c '.' > "$output_file" 
    exit 1
}

# Trap errors
trap 'error_handler $LINENO "$BASH_COMMAND"' ERR

# Exit immediately if a command exits with a non-zero status
set -e

# Read inputs
harbor_ca_cert_url=$(cat $inputs_file | jq -r .harbor_ca_cert_url)
harbor_host=$(cat $inputs_file | jq -r .harbor_host)
harbor_ip=$(cat $inputs_file | jq -r .harbor_ip)
harbor_password=$(cat $inputs_file | jq -r .harbor_password)
harbor_port=$(cat $inputs_file | jq -r .harbor_port)
harbor_project=$(cat $inputs_file | jq -r .harbor_project)
harbor_user=$(cat $inputs_file | jq -r .harbor_user)
source_images=$(cat $inputs_file | jq -r .source_images)

# Install docker, nslookup, and skopeo
echo "Install docker" | tee -a $logFile
apt-get update
apt-get install ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
echo "Install dnsutils (nslookup)" | tee -a $logFile
apt-get install -y dnsutils
echo "Install skopeo" | tee -a $logFile
apt-get install -y skopeo

# Test DNS resolution
if ! nslookup $harbor_host > /dev/null 2>&1; then
    echo "DNS resolution failed for $harbor_host. Adding to /etc/hosts." | tee -a $logFile
    echo "$harbor_ip $harbor_host" >> /etc/hosts
else
    echo "DNS resolution successful for $harbor_host." | tee -a $logFile
fi

# Configure harbor CA certificate
echo "Configuring harbor CA certificate"
if [ -n "$harbor_ca_cert_url" ]; then
    echo "Downloading harbor CA certificate" | tee -a $logFile
    curl -sSL $harbor_ca_cert_url -o /usr/local/share/ca-certificates/$harbor_host.crt
    update-ca-certificates
fi

# echo "docker login -u $harbor_user -p $harbor_password $harbor_host:$harbor_port" | tee -a $logFile
# docker login -u $harbor_user -p $harbor_password $harbor_host:$harbor_port

# for image in $(echo $source_images | jq -r '.[]'); do
#     echo "Processing image: $image" | tee -a $logFile
#     echo "docker pull $image" | tee -a $logFile
#     docker pull $image
#     remaining="${image#*/}"
#     echo "docker tag $image $harbor_host:$harbor_port/$harbor_project/$remaining" | tee -a $logFile
#     docker tag $image $harbor_host:$harbor_port/$harbor_project/$remaining
#     echo "docker push $harbor_host:$harbor_port/$harbor_project/$remaining" | tee -a $logFile
#     docker push $harbor_host:$harbor_port/$harbor_project/$remaining
#     echo "docker rmi $image" | tee -a $logFile
#     docker rmi $image
#     echo "docker rmi $harbor_host:$harbor_port/$harbor_project/$remaining" | tee -a $logFile
#     docker rmi $harbor_host:$harbor_port/$harbor_project/$remaining
# done

echo "skopeo login --username $harbor_user --password $harbor_password $harbor_host:$harbor_port" | tee -a $logFile
skopeo login --username $harbor_user --password $harbor_password $harbor_host:$harbor_port

for image in $(echo $source_images | jq -r '.[]'); do
    echo "Processing image: $image" | tee -a $logFile
    remaining="${image#*/}"
    echo "skopeo copy docker://$image docker://$harbor_host:$harbor_port/$harbor_project/$remaining" | tee -a $logFile
    skopeo copy docker://$image docker://$harbor_host:$harbor_port/$harbor_project/$remaining
done

# Cleanup
echo "docker logout $harbor_host:$harbor_port" | tee -a $logFile
skopeo logout $harbor_host:$harbor_port

# staging successful
echo "{\"status\":200}" | jq -c '.' > "$output_file" 
