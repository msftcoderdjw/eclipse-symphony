/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package configutils

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	configv1 "gopls-workspace/apis/config/v1"
	"gopls-workspace/constants"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	coacontexts "github.com/eclipse-symphony/symphony/coa/pkg/logger/contexts"
)

var (
	namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	configName    = os.Getenv(constants.ConfigName)
)

func GetValidationPoilicies() (map[string][]configv1.ValidationPolicy, error) {
	// home := homedir.HomeDir()
	// // use the current context in kubeconfig
	// config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	// if err != nil {
	// 	panic(err.Error())
	// }

	// // create the clientset
	// clientset, err := kubernetes.NewForConfig(config)

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	namespace, err := getNamespace()
	if err != nil {
		return nil, err
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), configName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var myConfig configv1.ProjectConfig
	data := configMap.Data["controller_manager_config.yaml"]
	err = yaml.Unmarshal([]byte(data), &myConfig)
	if err != nil {
		return nil, err
	}

	return myConfig.ValidationPolicies, nil
}
func getNamespace() (string, error) {
	// read the namespace from the file
	data, err := ioutil.ReadFile(namespaceFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func CheckValidationPack(myName string, myValue, validationType string, pack []configv1.ValidationStruct) (string, error) {
	if validationType == "unique" {
		for _, p := range pack {
			if p.Field == myValue {
				if myName != p.Name {
					return myValue, nil
				}
			}
		}
	}
	return "", nil
}

func PopulateActivityAndDiagnosticsContextFromAnnotations(objectId string, annotations map[string]string, operationName string, ctx context.Context, log logr.Logger) context.Context {
	correlationId := annotations[constants.AzureCorrelationIdKey]
	resourceId := annotations[constants.AzureResourceIdKey]
	location := annotations[constants.AzureLocationKey]
	systemData := annotations[constants.AzureSystemDataKey]

	// correlationId := uuid.New().String()
	// resourceId := objectId
	// location := "on-premise"
	// systemData := "{\"createdBy\":\"On-Premise\"}"

	resourceK8SId := objectId
	callerId := ""
	if systemData != "" {
		systemDataMap := make(map[string]string)
		if err := json.Unmarshal([]byte(systemData), &systemDataMap); err != nil {
			log.Info("Failed to unmarshal system data", "error", err)
		} else {
			// callerId = systemDataMap[constants.AzureCreatedByKey]
			callerId = "******"
		}
	}
	retCtx := coacontexts.PopulateResourceIdAndCorrelationIdToDiagnosticLogContext(correlationId, resourceId, ctx)
	retCtx = coacontexts.PatchActivityLogContextToCurrentContext(coacontexts.NewActivityLogContext(resourceId, location, operationName, correlationId, callerId, resourceK8SId), retCtx)
	return retCtx
}
