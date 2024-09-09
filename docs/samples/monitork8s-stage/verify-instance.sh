inputs_file=$1

output_file="${inputs_file%.*}-output.${inputs_file##*.}"

echo "{\"status\":200}" | jq -c '.' > "$output_file"