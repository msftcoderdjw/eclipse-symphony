go work use .
go build -tags=azure main.go 
kubectl cp main  workloadorchestration/symphony-api-85f7b675cb-57jjc:/main

kubectl exec -it symphony-api-85f7b675cb-57jjc -n workloadorchestration -c reids -- bash
redis-cli
HGETALL "DeployState.solution.symphony*cloudtestsite*target3a459a95fa44-v-st3a459a95fa47-v-testinsd" 