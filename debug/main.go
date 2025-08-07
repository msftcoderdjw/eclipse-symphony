package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gopls-workspace/constants"
	"gopls-workspace/utils"
	"reflect"
	"regexp"
	"strings"

	fabric_v1 "gopls-workspace/apis/fabric/v1"
	solution_v1 "gopls-workspace/apis/solution/v1"

	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/managers/solution"
	api_utils "github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/utils"

	sp "github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers"
	tgt "github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/target"

	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers/states"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers/states/redisstate"
	"github.com/eclipse-symphony/symphony/coa/pkg/logger"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// func main() {
// 	namespace := "test"
// 	var actionConfig *action.Configuration
// 	settings := cli.New()

// 	actionConfig = new(action.Configuration)
// 	actionConfig.Init(settings.RESTClientGetter(), namespace, "secret", func(format string, v ...interface{}) {
// 		fmt.Printf(format, v...)
// 	})

// 	listClient := action.NewList(actionConfig)
// 	listClient.Deployed = true
// 	var results []*release.Release
// 	results, _ = listClient.Run()

// 	releaseName := "test123"

// 	ret := make([]model.ComponentSpec, 0)
// 	for _, res := range results {
// 		if res.Name == releaseName {
// 			repo := ""
// 			name := ""
// 			if strings.HasPrefix(res.Chart.Metadata.Tags, "SYM-REPO:") { //we use this special metadata tag to remember the chart URL
// 				parts := strings.Split(res.Chart.Metadata.Tags, ";")
// 				if len(parts) != 2 {
// 					return
// 				}
// 				repo = parts[0][9:]
// 				name = parts[1][9:]
// 			}
// 			ret = append(ret, model.ComponentSpec{
// 				Name: "helmcomponent",
// 				Type: "helm.v3",
// 				Properties: map[string]interface{}{
// 					"releaseName": res.Name,
// 					"chart": map[string]string{
// 						"repo":    repo,
// 						"name":    name,
// 						"version": res.Chart.Metadata.Version,
// 					},
// 					"values": res.Config,
// 				},
// 			})
// 		}
// 	}

// 	retJson, _ := json.Marshal(ret)
// 	fmt.Println(string(retJson))

// 	changeProps := []model.PropertyDesc{
// 		{Name: "chart", IgnoreCase: false, SkipIfMissing: true}, //TODO: deep change detection on interface{}
// 		{Name: "values", PropChanged: propChange},
// 	}

// 	new := model.ComponentSpec{
// 		Name: "helmcomponent",
// 		Type: "helm.v3",
// 		Properties: map[string]interface{}{
// 			"releaseName": "test",
// 			"chart": map[string]string{
// 				"repo":    "bvtacr.azurecr.io/helm/simple-chart",
// 				"name":    "",
// 				"version": "0.4.0",
// 			},
// 			"values": map[string]interface{}{
// 				"key": "value",
// 			},
// 		},
// 	}

// 	deployment, err := deploymentBuilder(context.Background(), "target3a459a95fa44-v-st3a459a95fa47-v-testinsc", "cloudtestsite")
// 	if err != nil {
// 		fmt.Println("Error building deployment:", err)
// 		return
// 	}
// 	deploymentJson, _ := json.Marshal(*deployment)
// 	fmt.Println(string(deploymentJson))

// 	changed := IsComponentChanged(changeProps, ret[0], new)
// 	fmt.Println("Component changed:", changed)
// }

func main() {
	deployment, err := deploymentBuilder(context.Background(), "target3a459a95fa44-v-st3a459a95fa47-v-testinsd", "cloudtestsite")
	if err != nil {
		fmt.Println("Error building deployment:", err)
		return
	}
	reconcileCore(context.Background(), *deployment, false, "cloudtestsite", "target3a459a95fa44")
}

var (
	log = logger.NewLogger("coa.runtime")
)

const (
	SYMPHONY_AGENT string = "/symphony-agent:"
	ENV_NAME       string = "SYMPHONY_AGENT_ADDRESS"

	// DeploymentType_Update indicates the type of deployment is Update. This is
	// to give a deployment status on Symphony Target deployment.
	DeploymentType_Update string = "Target Update"
	// DeploymentType_Delete indicates the type of deployment is Delete. This is
	// to give a deployment status on Symphony Target deployment.
	DeploymentType_Delete string = "Target Delete"

	Summary         = "Summary"
	DeploymentState = "DeployState"
)

type SolutionManagerDeploymentState struct {
	Spec  model.DeploymentSpec  `json:"spec,omitempty"`
	State model.DeploymentState `json:"state,omitempty"`
}

func getDeploymentState(ctx context.Context, stateProvider states.IStateProvider, instance string, namespace string) *SolutionManagerDeploymentState {
	state, err := stateProvider.Get(ctx, states.GetRequest{
		ID: instance,
		Metadata: map[string]interface{}{
			"namespace": namespace,
			"group":     model.SolutionGroup,
			"version":   "v1",
			"resource":  DeploymentState,
		},
	})
	if err == nil {
		var managerState SolutionManagerDeploymentState
		jData, _ := json.Marshal(state.Body)
		log.InfofCtx(ctx, " M (Summary): previous state for instance %s in namespace %s: %s", instance, namespace, string(jData))
		err = json.Unmarshal(jData, &managerState)
		if err == nil {
			return &managerState
		}
	}
	return nil
}

func getCurrentApplicationScope(ctx context.Context, instance model.InstanceState, target model.TargetState) string {
	if instance.Spec.Scope == "" {
		if target.Spec.SolutionScope == "" {
			return "default"
		}
		return target.Spec.SolutionScope
	}
	return instance.Spec.Scope
}

func get(ctx context.Context, deployment model.DeploymentSpec, targetName string) (model.DeploymentState, []model.ComponentSpec, error) {
	ret := model.DeploymentState{}

	var err error
	var state model.DeploymentState
	state, err = solution.NewDeploymentState(deployment)
	if err != nil {
		return ret, nil, err
	}
	var plan model.DeploymentPlan
	plan, err = solution.PlanForDeployment(deployment, state)
	if err != nil {
		return ret, nil, err
	}
	ret = state
	ret.TargetComponent = make(map[string]string)
	retComponents := make([]model.ComponentSpec, 0)
	defaultScope := deployment.Instance.Spec.Scope

	for _, step := range plan.Steps {
		if targetName != "" && targetName != step.Target {
			continue
		}

		deployment.ActiveTarget = step.Target
		deployment.Instance.Spec.Scope = getCurrentApplicationScope(ctx, deployment.Instance, deployment.Targets[step.Target])

		var override tgt.ITargetProvider
		role := step.Role
		if role == "container" {
			role = "instance"
		}
		var provider providers.IProvider

		if override == nil {
			provider, err = sp.CreateProviderForTargetRole(&contexts.ManagerContext{}, step.Role, deployment.Targets[step.Target], override)
			if err != nil {
				return ret, nil, err
			}
		} else {
			provider = override
		}
		var components []model.ComponentSpec
		components, err = (provider.(tgt.ITargetProvider)).Get(ctx, deployment, step.Components)

		if err != nil {
			return ret, nil, err
		}
		for _, c := range components {
			key := fmt.Sprintf("%s::%s", c.Name, step.Target)
			role := c.Type
			if role == "" {
				role = "container"
			}
			ret.TargetComponent[key] = role
			found := false
			for _, rc := range retComponents {
				if rc.Name == c.Name {
					found = true
					break
				}
			}
			if !found {
				retComponents = append(retComponents, c)
			}
		}
		deployment.Instance.Spec.Scope = defaultScope
	}
	ret.Components = retComponents
	return ret, retComponents, nil
}

func logByJson(v any, prompt string) {
	jData, _ := json.Marshal(v)
	log.Infof("%s: %s", prompt, string(jData))
}

func logByJsonWithCtx(ctx context.Context, v any, prompt string) {
	jData, _ := json.Marshal(v)
	log.InfofCtx(ctx, "%s: %s", prompt, string(jData))
}

func canSkipStep(ctx context.Context, step model.DeploymentStep, target string, provider tgt.ITargetProvider, previousComponents []model.ComponentSpec, currentState model.DeploymentState) bool {
	logByJsonWithCtx(ctx, step, "arg: step")
	logByJsonWithCtx(ctx, target, "arg: target")
	logByJsonWithCtx(ctx, previousComponents, "arg: previousComponents")
	logByJsonWithCtx(ctx, currentState, "arg: currentState")
	for _, newCom := range step.Components {
		key := fmt.Sprintf("%s::%s", newCom.Component.Name, target)
		if newCom.Action == model.ComponentDelete {
			for _, c := range previousComponents {
				if c.Name == newCom.Component.Name && currentState.TargetComponent[key] != "" {
					log.InfoCtx(ctx, "debug 1")
					return false // current component still exists, desired is to remove it. The step can't be skipped
				}
			}

		} else {
			found := false
			for _, c := range previousComponents {
				if c.Name == newCom.Component.Name && currentState.TargetComponent[key] != "" && !strings.HasPrefix(currentState.TargetComponent[key], "-") {
					found = true
					rule := provider.GetValidationRule(ctx)
					for _, sc := range currentState.Components {
						if sc.Name == c.Name {
							logByJsonWithCtx(ctx, c, "previous component")
							logByJsonWithCtx(ctx, newCom.Component, "new component")
							logByJsonWithCtx(ctx, sc, "current component")

							previousAndNew := rule.IsComponentChanged(c, newCom.Component)
							currentAndNew := rule.IsComponentChanged(sc, newCom.Component)
							log.InfofCtx(ctx, "debug: previousAndNew: %v, currentAndNew: %v", previousAndNew, currentAndNew)
							if previousAndNew || currentAndNew {
								log.InfoCtx(ctx, "debug 2")
								return false // component has changed, can't skip the step
							}
							break
						}
					}
					break
				}
			}
			if !found {
				log.InfoCtx(ctx, "debug 3")
				return false //current component doesn't exist, desired is to update it. The step can't be skipped
			}
		}
	}
	log.InfoCtx(ctx, "debug 4")
	return true
}

func reconcileCore(ctx context.Context, deployment model.DeploymentSpec, remove bool, namespace string, targetName string) error {
	stateProvider := &redisstate.RedisStateProvider{}
	stateProvider.Init(redisstate.RedisStateProviderConfig{
		Host: "localhost:6379",
	})

	if deployment.IsInActive {
		remove = true
	}
	var err error
	summary := model.SummarySpec{
		TargetResults:       make(map[string]model.TargetResultSpec),
		TargetCount:         len(deployment.Targets),
		SuccessCount:        0,
		AllAssignedDeployed: false,
		JobID:               deployment.JobID,
	}

	summary.IsRemoval = remove
	summaryId := deployment.Instance.ObjectMeta.GetSummaryId()
	if summaryId == "" {
		return err
	}

	previousDesiredState := getDeploymentState(ctx, stateProvider, deployment.Instance.ObjectMeta.Name, namespace)

	var currentDesiredState, currentState model.DeploymentState
	currentDesiredState, err = solution.NewDeploymentState(deployment)
	if err != nil {
		return err
	}
	currentState, _, err = get(ctx, deployment, targetName)
	if err != nil {
		return err
	}
	desiredState := currentDesiredState
	if previousDesiredState != nil {
		desiredState = solution.MergeDeploymentStates(&previousDesiredState.State, currentDesiredState)
	}

	if remove {
		desiredState.MarkRemoveAll()
	}

	mergedState := solution.MergeDeploymentStates(&currentState, desiredState)
	var plan model.DeploymentPlan
	plan, err = solution.PlanForDeployment(deployment, mergedState)
	if err != nil {
		return err
	}

	col := api_utils.MergeCollection(deployment.Solution.Spec.Metadata, deployment.Instance.Spec.Metadata)
	dep := deployment
	dep.Instance.Spec.Metadata = col
	targetResult := make(map[string]int)

	summary.PlannedDeployment = 0
	for _, step := range plan.Steps {
		summary.PlannedDeployment += len(step.Components)
	}
	summary.CurrentDeployed = 0

	plannedCount := 0
	planSuccessCount := 0
	for _, step := range plan.Steps {
		for _, component := range step.Components {
			log.DebugfCtx(ctx, " M (Solution): processing component %s with action %s", component.Component.Name, component.Action)
		}

		if targetName != "" && targetName != step.Target {
			continue
		}

		plannedCount++

		dep.ActiveTarget = step.Target
		var override tgt.ITargetProvider
		role := step.Role
		if role == "container" {
			role = "instance"
		}
		var provider providers.IProvider
		if override == nil {
			targetSpec := getTargetStateForStep(step, deployment, previousDesiredState)
			provider, err = sp.CreateProviderForTargetRole(&contexts.ManagerContext{}, step.Role, targetSpec, override)
			if err != nil {
				log.ErrorfCtx(ctx, " M (Solution): failed to create provider: %+v", err)
				return err
			}
		} else {
			provider = override
		}
		var componentResults = make(map[string]model.ComponentResultSpec)
		if previousDesiredState != nil {
			testState := solution.MergeDeploymentStates(&previousDesiredState.State, currentState)
			if canSkipStep(ctx, step, step.Target, provider.(tgt.ITargetProvider), previousDesiredState.State.Components, testState) {
				summary.UpdateTargetResult(step.Target, model.TargetResultSpec{Status: "OK", Message: "", ComponentResults: componentResults})
				log.InfofCtx(ctx, " M (Solution): skipping step with role %s on target %s", step.Role, step.Target)
				targetResult[step.Target] = 1
				planSuccessCount++
				summary.CurrentDeployed += len(step.Components)
				continue
			} else {
				log.InfofCtx(ctx, " M (Solution): step with role %s on target %s is not skipped", step.Role, step.Target)
			}
		}
	}
	return nil
}

// The deployment spec may have changed, so the previous target is not in the new deployment anymore
func getTargetStateForStep(step model.DeploymentStep, deployment model.DeploymentSpec, previousDeploymentState *SolutionManagerDeploymentState) model.TargetState {
	//first find the target spec in the deployment
	targetSpec, ok := deployment.Targets[step.Target]
	if !ok {
		if previousDeploymentState != nil {
			targetSpec = previousDeploymentState.Spec.Targets[step.Target]
		}
	}
	return targetSpec
}

func getInstanceGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "solution.symphony",
		Version:  "v1",
		Resource: "instances",
	}
}

func getSolutionGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "solution.symphony",
		Version:  "v1",
		Resource: "solutions",
	}
}

func getTargetGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "fabric.symphony",
		Version:  "v1",
		Resource: "targets",
	}
}

func getInstance(ctx context.Context, dynClient dynamic.Interface, name string, namespace string) (*solution_v1.Instance, error) {
	item, err := dynClient.Resource(getInstanceGVR()).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	instance := &solution_v1.Instance{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func listTargets(ctx context.Context, dynClient dynamic.Interface, namespace string) (*fabric_v1.TargetList, error) {
	targetList := &fabric_v1.TargetList{}
	item, err := dynClient.Resource(getTargetGVR()).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), targetList)
	if err != nil {
		return nil, err
	}
	return targetList, nil
}

func getSolution(ctx context.Context, dynClient dynamic.Interface, name string, namespace string) (*solution_v1.Solution, error) {
	item, err := dynClient.Resource(getSolutionGVR()).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	solution := &solution_v1.Solution{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), solution)
	if err != nil {
		return nil, err
	}
	return solution, nil
}

func deploymentBuilder(ctx context.Context, instanceName string, namespace string) (*model.DeploymentSpec, error) {
	var kConfig *rest.Config
	// kConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	kConfig, err := rest.InClusterConfig()
	dynClient, err := dynamic.NewForConfig(kConfig)
	var deployment model.DeploymentSpec

	var instance *solution_v1.Instance
	instance, err = getInstance(ctx, dynClient, instanceName, namespace)
	if err != nil {
		return nil, err
	}

	deploymentResources := &utils.DeploymentResources{
		Instance:         *instance,
		Solution:         solution_v1.Solution{},
		TargetList:       fabric_v1.TargetList{},
		TargetCandidates: []fabric_v1.Target{},
	}

	solutionName := api_utils.ConvertReferenceToObjectName(instance.Spec.Solution)
	solution, err := getSolution(ctx, dynClient, solutionName, instance.Namespace)
	if err != nil {
		return nil, v1alpha2.NewCOAError(err, "failed to get solution", v1alpha2.SolutionGetFailed)
	}
	deploymentResources.Solution = *solution
	// Get targets
	targetList, err := listTargets(ctx, dynClient, instance.Namespace)
	if err != nil {
		return nil, v1alpha2.NewCOAError(err, "failed to list targets", v1alpha2.TargetListGetFailed)
	}
	deploymentResources.TargetList = *targetList

	// Get target candidates
	deploymentResources.TargetCandidates = utils.MatchTargets(*instance, deploymentResources.TargetList)
	if len(deploymentResources.TargetCandidates) == 0 {
		_ = v1alpha2.NewCOAError(nil, "no target candidates found", v1alpha2.TargetCandidatesNotFound)
	}

	deployment, err = utils.CreateSymphonyDeployment(ctx, *instance, deploymentResources.Solution, deploymentResources.TargetCandidates, instance.GetNamespace())
	deployment.JobID = instance.GetAnnotations()[constants.SummaryJobIdKey]
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

func convertMapStringToStringInterface(m map[string]string) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func IsComponentChanged(changeDetectionProperties []model.PropertyDesc, old model.ComponentSpec, new model.ComponentSpec) bool {
	if detectChanges(changeDetectionProperties, old.Name, new.Name, old.Properties, new.Properties) {
		return true
	}
	if detectChanges(changeDetectionProperties, old.Name, new.Name,
		convertMapStringToStringInterface(old.Metadata),
		convertMapStringToStringInterface(new.Metadata)) {
		return true
	}
	return false
}

func detectChanges(properties []model.PropertyDesc, oldName string, newName string, oldValues map[string]interface{}, newValues map[string]interface{}) bool {
	// loop all provider's change detection properties
	for _, p := range properties {
		if strings.Contains(p.Name, "*") {
			escapedPattern := regexp.QuoteMeta(p.Name)
			// Replace the wildcard (*) with a regular expression pattern
			regexpPattern := strings.ReplaceAll(escapedPattern, `\*`, ".*")
			// Compile the regular expression
			regexpObject := regexp.MustCompile("^" + regexpPattern + "$")
			mergedKeys := mergeKeysInOldAndNew(oldValues, newValues)
			for _, k := range mergedKeys {
				if regexpObject.MatchString(k) {
					if compareProperties(p, oldValues, newValues, k) {
						return true
					}
				}
			}
		} else {
			if p.IsComponentName {
				if !compareStrings(oldName, newName, p.IgnoreCase, p.PrefixMatch) {
					return true
				}
			} else {
				if compareProperties(p, oldValues, newValues, p.Name) {
					return true
				}
			}
		}
	}

	return false
}

func mergeKeysInOldAndNew(oldValues map[string]interface{}, newValues map[string]interface{}) []string {
	keys := make(map[string]bool)
	for k := range oldValues {
		keys[k] = true
	}
	for k := range newValues {
		keys[k] = true
	}
	mergedKeys := make([]string, 0, len(keys))
	for k := range keys {
		mergedKeys = append(mergedKeys, k)
	}
	return mergedKeys
}

func isEmpty(values interface{}) bool {
	if values == nil {
		return true
	}
	valueMap, ok := values.(map[string]interface{})
	if ok {
		return len(valueMap) == 0
	}
	return false
}

func propChange(old, new interface{}) bool {
	// scenarios where either is an empty map and the other is nil count as no change
	if isEmpty(old) && isEmpty(new) {
		return false
	}
	return !reflect.DeepEqual(old, new)
}

func compareStrings(a, b string, ignoreCase bool, prefixMatch bool) bool {
	ta := a
	tb := b
	if ignoreCase {
		ta = strings.ToLower(a)
		tb = strings.ToLower(b)
	}
	if !prefixMatch {
		return ta == tb
	} else {
		return strings.HasPrefix(tb, ta) || strings.HasPrefix(ta, tb)
	}
}

func compareProperties(c model.PropertyDesc, old map[string]interface{}, new map[string]interface{}, key string) bool {
	v, ook := old[key]
	nv, nok := new[key]
	if c.PropChanged != nil {
		return c.PropChanged(v, nv)
	}

	if ook && nok {
		// case 1: key exists in both old and new
		// compare the values, if different, return true
		if !compareStrings(fmt.Sprintf("%v", v), fmt.Sprintf("%v", nv), c.IgnoreCase, c.PrefixMatch) {
			return true
		} else {
			return false
		}
	} else if !ook && !nok {
		// case 2: key does not exist in both old and new
		// return false to indicate no change
		return false
	} else {
		// case 3: one of them is missing
		// if the property is optional, no matter it doesn't exist in old or new, return false to indicate no change
		if !c.SkipIfMissing {
			return true
		} else {
			return false
		}
	}
}
