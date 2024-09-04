az acr login -n toolchainorchestratordev
cd ~/github/dailygo/helm/my-app-config
docker build -t my-app-config:v2.0 .
docker tag my-app-config:v2.0 toolchainorchestratordev.azurecr.io/my-app-config:v2.0
docker push toolchainorchestratordev.azurecr.io/my-app-config:v2.0

cd ~/github/dailygo/helm
helm install my-app-config ./my-app-config-chart
helm uninstall my-app-config

helm upgrade --install my-app-config ./my-app-config-chart --set image.tag=v2.0
helm upgrade my-app-config ./my-app-config-chart -f ./my-app-config-chart/new-values.yaml

kubectl get deployment my-app-config -n default -o=jsonpath='{.spec.template.spec.containers[*].image}'
kubectl get pods -l app=my-app-config -n default -o=jsonpath='{range .items[*]}{.metadata.name}: {.spec.containers[*].image} - Status: {.status.phase}{"\n"}{end}'

kubectl logs 'deployment/my-app-config' --all-containers -n default
kubectl exec -it 'deployment/my-app-config' -- sh

kubectl apply -f /home/jiadu/github/dailygo/helm/test-app-config-chart-prepare/my-config.yaml -n default2
kubectl apply -f /home/jiadu/github/dailygo/helm/test-app-config-chart-prepare/my-env-config.yaml -n default2
helm install test-app-config ./test-app-config-chart --set config.myConfigName=my-config --set config.myEnvConfigName=my-env-config --set config.myEnvConfigAnnotation=my-env-config-v1 -n default2
helm upgrade test-app-config ./test-app-config-chart --set config.myConfigName=my-config --set config.myEnvConfigName=my-env-config --set config.myEnvConfigAnnotation=my-env-config-v1 -n default2

helm uninstall test-app-config -n default2

helm upgrade test-app-config ./test-app-config-chart -f ./test-app-config-chart/new-values.yaml

cd ~/github/dailygo/helm/test-app-config-chart
helm package .
helm push .\test-app-config-chart-0.1.0.tgz oci://toolchainorchestratordev.azurecr.io/helm
helm push .\my-app-config-chart-0.1.0.tgz oci://toolchainorchestratordev.azurecr.io/helm