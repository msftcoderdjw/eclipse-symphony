/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package materialize

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/metrics"
	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/stage"
	api_utils "github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/utils"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability"
	observ_utils "github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/eclipse-symphony/symphony/coa/pkg/logger"
)

const (
	loggerName   = "providers.stage.materialize"
	providerName = "P (Materialize Stage)"
	materialize  = "materialize"
)

var (
	maLock                   sync.Mutex
	mLog                     = logger.NewLogger(loggerName)
	once                     sync.Once
	providerOperationMetrics *metrics.Metrics
)

type MaterializeStageProviderConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type MaterializeStageProvider struct {
	Config    MaterializeStageProviderConfig
	Context   *contexts.ManagerContext
	ApiClient api_utils.ApiClient
}

func (s *MaterializeStageProvider) Init(config providers.IProviderConfig) error {
	ctx, span := observability.StartSpan("[Stage] Materialize Provider", context.TODO(), &map[string]string{
		"method": "Init",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)
	defer observ_utils.EmitUserDiagnosticsLogs(ctx, &err)

	maLock.Lock()
	defer maLock.Unlock()
	var mockConfig MaterializeStageProviderConfig
	mockConfig, err = toMaterializeStageProviderConfig(config)
	if err != nil {
		return err
	}
	s.Config = mockConfig
	s.ApiClient, err = api_utils.GetApiClient()
	if err != nil {
		return err
	}
	once.Do(func() {
		if providerOperationMetrics == nil {
			providerOperationMetrics, err = metrics.New()
			if err != nil {
				mLog.ErrorfCtx(ctx, "  P (Materialize Stage): failed to create metrics: %+v", err)
			}
		}
	})
	return err
}
func (s *MaterializeStageProvider) SetContext(ctx *contexts.ManagerContext) {
	s.Context = ctx
}
func toMaterializeStageProviderConfig(config providers.IProviderConfig) (MaterializeStageProviderConfig, error) {
	ret := MaterializeStageProviderConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(data, &ret)
	return ret, err
}
func (i *MaterializeStageProvider) InitWithMap(properties map[string]string) error {
	config, err := MaterialieStageProviderConfigFromMap(properties)
	if err != nil {
		return err
	}
	return i.Init(config)
}
func MaterializeStageProviderConfigFromVendorMap(properties map[string]string) (MaterializeStageProviderConfig, error) {
	ret := make(map[string]string)
	for k, v := range properties {
		if strings.HasPrefix(k, "wait.") {
			ret[k[5:]] = v
		}
	}
	return MaterialieStageProviderConfigFromMap(ret)
}
func MaterialieStageProviderConfigFromMap(properties map[string]string) (MaterializeStageProviderConfig, error) {
	ret := MaterializeStageProviderConfig{}
	if api_utils.ShouldUseUserCreds() {
		user, err := api_utils.GetString(properties, "user")
		if err != nil {
			return ret, err
		}
		ret.User = user
		if ret.User == "" {
			return ret, v1alpha2.NewCOAError(nil, "user is required", v1alpha2.BadConfig)
		}
		password, err := api_utils.GetString(properties, "password")
		if err != nil {
			return ret, err
		}
		ret.Password = password
	}
	return ret, nil
}
func (i *MaterializeStageProvider) Process(ctx context.Context, mgrContext contexts.ManagerContext, inputs map[string]interface{}) (map[string]interface{}, bool, error) {
	ctx, span := observability.StartSpan("[Stage] Materialize Provider", ctx, &map[string]string{
		"method": "Process",
	})
	var err error = nil
	defer observ_utils.CloseSpanWithError(span, &err)
	defer observ_utils.EmitUserDiagnosticsLogs(ctx, &err)
	mLog.InfoCtx(ctx, "  P (Materialize Processor): processing inputs")
	processTime := time.Now().UTC()
	functionName := observ_utils.GetFunctionName()

	outputs := make(map[string]interface{})

	objects, ok := inputs["names"].([]interface{})
	if !ok {
		err = v1alpha2.NewCOAError(nil, "input names is not a valid list", v1alpha2.BadRequest)
		providerOperationMetrics.ProviderOperationErrors(
			materialize,
			functionName,
			metrics.ProcessOperation,
			metrics.ValidateOperationType,
			v1alpha2.BadConfig.String(),
		)
		return outputs, false, err
	}
	prefixedNames := make([]string, len(objects))
	for i, object := range objects {
		objString, ok := object.(string)
		if !ok {
			err = v1alpha2.NewCOAError(nil, fmt.Sprintf("input name is not a valid string: %v", objects), v1alpha2.BadRequest)
			providerOperationMetrics.ProviderOperationErrors(
				materialize,
				functionName,
				metrics.ProcessOperation,
				metrics.ValidateOperationType,
				v1alpha2.BadConfig.String(),
			)
			return outputs, false, err
		}
		if s, ok := inputs["__origin"]; ok {
			prefixedNames[i] = fmt.Sprintf("%s-%s", s, objString)
		} else {
			prefixedNames[i] = objString
		}
	}
	namespace := stage.GetNamespace(inputs)
	if namespace == "" {
		namespace = "default"
	}

	mLog.DebugfCtx(ctx, "  P (Materialize Processor): masterialize %v in namespace %s", prefixedNames, namespace)

	var catalogs []model.CatalogState
	catalogs, err = i.ApiClient.GetCatalogs(ctx, namespace, i.Config.User, i.Config.Password)

	if err != nil {
		mLog.ErrorfCtx(ctx, "Failed to get catalogs: %s", err.Error())
		providerOperationMetrics.ProviderOperationErrors(
			materialize,
			functionName,
			metrics.ProcessOperation,
			metrics.RunOperationType,
			v1alpha2.CatalogsGetFailed.String(),
		)
		return outputs, false, err
	}
	creationCount := 0
	for _, catalog := range catalogs {
		for _, object := range prefixedNames {
			object := api_utils.ReplaceSeperator(object)
			if catalog.ObjectMeta.Name == object {
				objectData, _ := json.Marshal(catalog.Spec.Properties) //TODO: handle errors
				name := catalog.ObjectMeta.Name
				if s, ok := inputs["__origin"]; ok {
					name = strings.TrimPrefix(catalog.ObjectMeta.Name, fmt.Sprintf("%s-", s))
				}
				switch catalog.Spec.Type {
				case "instance":
					var instanceState model.InstanceState
					err = json.Unmarshal(objectData, &instanceState)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to unmarshal instance state for catalog %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.InvalidInstanceCatalog.String(),
						)
						return outputs, false, err
					}
					// If inner instace defines a display name, use it as the name
					if instanceState.Spec.DisplayName != "" {
						instanceState.ObjectMeta.Name = instanceState.Spec.DisplayName
					}
					instanceState.ObjectMeta = updateObjectMeta(instanceState.ObjectMeta, inputs, name)
					objectData, _ := json.Marshal(instanceState)
					mLog.DebugfCtx(ctx, "  P (Materialize Processor): materialize instance %v to namespace %s", instanceState.ObjectMeta.Name, instanceState.ObjectMeta.Namespace)
					observ_utils.EmitUserAuditsLogs(ctx, "  P (Materialize Processor): Start to materialize instance %v to namespace %s", instanceState.ObjectMeta.Name, instanceState.ObjectMeta.Namespace)
					err = i.ApiClient.CreateInstance(ctx, instanceState.ObjectMeta.Name, objectData, instanceState.ObjectMeta.Namespace, i.Config.User, i.Config.Password)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to create instance %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.CreateInstanceFromCatalogFailed.String(),
						)
						return outputs, false, err
					}
					creationCount++
				case "solution":
					var solutionState model.SolutionState
					err = json.Unmarshal(objectData, &solutionState)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to unmarshal solution state for catalog %s: %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.InvalidSolutionCatalog.String(),
						)
						return outputs, false, err
					}
					// If inner solution defines a display name, use it as the name
					if solutionState.Spec.DisplayName != "" {
						solutionState.ObjectMeta.Name = solutionState.Spec.DisplayName
					}
					solutionState.ObjectMeta = updateObjectMeta(solutionState.ObjectMeta, inputs, name)
					objectData, _ := json.Marshal(solutionState)
					mLog.DebugfCtx(ctx, "  P (Materialize Processor): materialize solution %v to namespace %s", solutionState.ObjectMeta.Name, solutionState.ObjectMeta.Namespace)
					observ_utils.EmitUserAuditsLogs(ctx, "  P (Materialize Processor): Start to materialize solution %v to namespace %s", solutionState.ObjectMeta.Name, solutionState.ObjectMeta.Namespace)
					err = i.ApiClient.UpsertSolution(ctx, solutionState.ObjectMeta.Name, objectData, solutionState.ObjectMeta.Namespace, i.Config.User, i.Config.Password)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to create solution %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.CreateSolutionFromCatalogFailed.String(),
						)
						return outputs, false, err
					}
					creationCount++
				case "target":
					var targetState model.TargetState
					err = json.Unmarshal(objectData, &targetState)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to unmarshal target state for catalog %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.InvalidTargetCatalog.String(),
						)
						return outputs, false, err
					}
					// If inner target defines a display name, use it as the name
					if targetState.Spec.DisplayName != "" {
						targetState.ObjectMeta.Name = targetState.Spec.DisplayName
					}
					targetState.ObjectMeta = updateObjectMeta(targetState.ObjectMeta, inputs, name)
					objectData, _ := json.Marshal(targetState)
					mLog.DebugfCtx(ctx, "  P (Materialize Processor): materialize target %v to namespace %s", targetState.ObjectMeta.Name, targetState.ObjectMeta.Namespace)
					observ_utils.EmitUserAuditsLogs(ctx, "  P (Materialize Processor): Start to materialize target %v to namespace %s", targetState.ObjectMeta.Name, targetState.ObjectMeta.Namespace)
					err = i.ApiClient.CreateTarget(ctx, targetState.ObjectMeta.Name, objectData, targetState.ObjectMeta.Namespace, i.Config.User, i.Config.Password)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to create target %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.CreateTargetFromCatalogFailed.String(),
						)
						return outputs, false, err
					}
					creationCount++
				default:
					// Check wrapped catalog structure and extract wrapped catalog name
					var catalogState model.CatalogState
					err = json.Unmarshal(objectData, &catalogState)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to unmarshal catalog state for catalog %s: %s", name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.InvalidCatalogCatalog.String(),
						)
						return outputs, false, err
					}
					catalogState.ObjectMeta = updateObjectMeta(catalogState.ObjectMeta, inputs, name)
					objectData, _ := json.Marshal(catalogState)
					mLog.DebugfCtx(ctx, "  P (Materialize Processor): materialize catalog %v to namespace %s", catalogState.ObjectMeta.Name, catalogState.ObjectMeta.Namespace)
					observ_utils.EmitUserAuditsLogs(ctx, "  P (Materialize Processor): Start to materialize catalog %v to namespace %s", catalogState.ObjectMeta.Name, catalogState.ObjectMeta.Namespace)
					err = i.ApiClient.UpsertCatalog(ctx, catalogState.ObjectMeta.Name, objectData, i.Config.User, i.Config.Password)
					if err != nil {
						mLog.ErrorfCtx(ctx, "Failed to create catalog %s: %s", catalogState.ObjectMeta.Name, err.Error())
						providerOperationMetrics.ProviderOperationErrors(
							materialize,
							functionName,
							metrics.ProcessOperation,
							metrics.RunOperationType,
							v1alpha2.CreateCatalogFromCatalogFailed.String(),
						)
						return outputs, false, err
					}
					creationCount++
				}
			}
		}
	}
	if creationCount < len(objects) {
		err = v1alpha2.NewCOAError(nil, "failed to create all objects", v1alpha2.InternalError)
		providerOperationMetrics.ProviderOperationErrors(
			materialize,
			functionName,
			metrics.ProcessOperation,
			metrics.RunOperationType,
			v1alpha2.MaterializeBatchFailed.String(),
		)
		return outputs, false, err
	}
	providerOperationMetrics.ProviderOperationLatency(
		processTime,
		materialize,
		metrics.ProcessOperation,
		metrics.RunOperationType,
		functionName,
	)
	return outputs, false, nil
}

func updateObjectMeta(objectMeta model.ObjectMeta, inputs map[string]interface{}, catalogName string) model.ObjectMeta {
	if objectMeta.Name == "" {
		// use the same name as catalog wrapping it if not provided
		objectMeta.Name = catalogName
	}
	// stage inputs override objectMeta namespace
	if s := stage.GetNamespace(inputs); s != "" {
		objectMeta.Namespace = s
	} else if objectMeta.Namespace == "" {
		objectMeta.Namespace = "default"
	}
	return objectMeta
}
