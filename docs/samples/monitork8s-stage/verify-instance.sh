#!/bin/bash
##
## Copyright (c) Microsoft Corporation.
## Licensed under the MIT license.
## SPDX-License-Identifier: MIT
##

inputs_file=$1

output_file="${inputs_file%.*}-output.${inputs_file##*.}"

instance_name=$(jq -r '.instance_name' < "$inputs_file")
instance=$(jq -r ".items[] | select(.metadata.name == \"$instance_name\")" "$inputs_file")

echo "instance: $instance"

# get the instance object
status=$(echo $instance | jq -r '.status.provisioningStatus.status')
if [ "$status" == "Succeeded" ]; then
    echo "{\"status\":200}" | jq -c '.' > "$output_file"
elif [ "$status" == "Failed" ]; then
    echo "{\"status\":500}" | jq -c '.' > "$output_file"
else
    echo "{\"status\":202}" | jq -c '.' > "$output_file"
fi