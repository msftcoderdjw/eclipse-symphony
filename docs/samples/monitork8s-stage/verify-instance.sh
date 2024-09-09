#!/bin/bash
##
## Copyright (c) Microsoft Corporation.
## Licensed under the MIT license.
## SPDX-License-Identifier: MIT
##

inputs_file=$1

output_file="${inputs_file%.*}-output.${inputs_file##*.}"

items=$(jq -r '.items[]' < "$inputs_file")
instance_name=$(jq -r '.instanceName' < "$inputs_file")

# get the instance object
for item in $items; do
    status=$item.status.provisioningStatus.status
    if [ "$status" == "Succeeded" ]; then
        echo "{\"status\":200}" | jq -c '.' > "$output_file"
        exit 0
    elif [ "$status" == "Failed" ]; then
        echo "{\"status\":500}" | jq -c '.' > "$output_file"
        exit 0
    fi
done

echo "{\"status\":202}" | jq -c '.' > "$output_file"