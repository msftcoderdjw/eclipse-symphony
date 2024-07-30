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
    echo "Error occurred in script at line: $1" | tee -a $logFile
    echo "{\"status\":500}" | jq -c '.' > "$output_file" 
    exit 1
}

# Trap errors
trap 'error_handler $LINENO' ERR

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

# Install docker and nslookup
echo "Install docker and nslookup" | tee -a $logFile
apt-get update
apt-get install -y docker.io dnsutils

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

echo "docker login -u $harbor_user -p $harbor_password $harbor_host:$harbor_port" | tee -a $logFile

for image in $(echo $source_images | jq -r '.[]'); do
    echo "Processing image: $image" | tee -a $logFile
    echo "docker pull $image" | tee -a $logFile
    docker pull $image
    remaing="${image#*/}"
    echo "docker tag $image $harbor_host:$harbor_port/$harbor_project/$remaing" | tee -a $logFile
    docker tag $image $harbor_host:$harbor_port/$harbor_project/$remaing
    echo "docker push $harbor_host:$harbor_port/$harbor_project/$remaing" | tee -a $logFile
    docker push $harbor_host:$harbor_port/$harbor_project/$remaing
    echo "docker rmi $image" | tee -a $logFile
    docker rmi $image
    echo "docker rmi $harbor_host:$harbor_port/$harbor_project/$remaing" | tee -a $logFile
    docker rmi $harbor_host:$harbor_port/$harbor_project/$remaing
done

# staging successful
echo "{\"status\":200}" | jq -c '.' > "$output_file" 
