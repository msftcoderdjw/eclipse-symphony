# install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
kubectl version

inputs_file=$1

output_file="${inputs_file%.*}-output.${inputs_file##*.}"
# polling instance objects

# sleep 10 before instance reconcile pick solution changes
sleep 10

# get the instance name
instance_name=$(jq -r '.instanceName' < "$inputs_file")
namespace_name=$(jq -r '.namespace' < "$inputs_file")
max_retries=$(jq -r '.maxRetries' < "$inputs_file")
# get the instance object
instance=$(kubectl get instance -n "$namespace_name" "$instance_name" -o json)
status=$(echo $instance | jq -r '.status.provisioningStatus.status')

# max_retries is optional, if not provided, default to 10
if [ -z "$max_retries" ]; then
    max_retries=10
fi

loop_count=0
while [ "$status" != "Succeeded" ] && [ "$status" != "Failed" ] && [ $loop_count -lt max_retries ]; do
    sleep 10
    instance=$(kubectl get instance -n "$namespace_name" "$instance_name" -o json)
    status=$(echo $instance | jq -r '.status.provisioningStatus.status')
    loop_count=$((loop_count+1))
done

if [ "$status" == "Succeeded" ]; then
    echo "{\"status\":200}" | jq -c '.' > "$output_file"
elif [ "$status" == "Failed" ]; then
    echo "{\"status\":500}" | jq -c '.' > "$output_file"
else
    echo "{\"status\":202}" | jq -c '.' > "$output_file"
fi