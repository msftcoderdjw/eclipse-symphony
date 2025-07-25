/*
* Copyright (c) Microsoft Corporation.
* Licensed under the MIT license.
* SPDX-License-Identifier: MIT
 */

package helm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/target/conformance"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/postrender"
	"k8s.io/client-go/rest"
)

const (
	odooRepo         = "registry-1.docker.io/bitnamicharts/odoo"
	odooVersion      = "28.1.1"
	defaultTestScope = "alice-springs"
)

// TestHelmTargetProviderConfigFromMapNil tests the HelmTargetProviderConfigFromMap function with nil input
func TestHelmTargetProviderConfigFromMapNil(t *testing.T) {
	_, err := HelmTargetProviderConfigFromMap(nil)
	assert.Nil(t, err)
}

// TestHelmTargetProviderConfigFromMapEmpty tests the HelmTargetProviderConfigFromMap function with empty input
func TestHelmTargetProviderConfigFromMapEmpty(t *testing.T) {
	_, err := HelmTargetProviderConfigFromMap(map[string]string{})
	assert.Nil(t, err)
}

// TestHelmTargetProviderInitEmptyConfig tests the Init function of HelmTargetProvider with empty config
func TestHelmTargetProviderInitEmptyConfig(t *testing.T) {
	config := HelmTargetProviderConfig{}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.NotNil(t, err)
}

func TestInitWithMap(t *testing.T) {
	configMap := map[string]string{
		"name":       "test",
		"configType": "bytes",
		"inCluster":  "false",
		"configData": "data",
		"context":    "context",
	}
	provider := HelmTargetProvider{}
	err := provider.InitWithMap(configMap)
	assert.NotNil(t, err)

	configMap = map[string]string{
		"name":       "test",
		"configType": "bytes",
		"inCluster":  "false",
		"context":    "context",
	}
	err = provider.InitWithMap(configMap)
	assert.NotNil(t, err)

	configMap = map[string]string{
		"name":       "test",
		"configType": "wrongtype",
		"inCluster":  "false",
		"context":    "context",
	}
	err = provider.InitWithMap(configMap)
	assert.NotNil(t, err)
}

// TestHelmTargetProviderGetHelmProperty tests the getHelmValuesFromComponent function with valid input
func TestHelmTargetProviderGetHelmPropertyMissingRepo(t *testing.T) {
	_, err := getHelmPropertyFromComponent(model.ComponentSpec{
		Name: "odoo",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo":    "", // blank repo
				"name":    "odoo",
				"version": "0.1.1",
			},
			"values": map[string]interface{}{
				"service.type": "NodePort",
				"service.port": 80,
			},
		},
	})
	assert.NotNil(t, err)
}

func TestHelmTargetProviderGetHelmProperty(t *testing.T) {
	_, err := getHelmPropertyFromComponent(model.ComponentSpec{
		Name: "odoo",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo":    odooRepo,
				"version": odooVersion,
			},
			"values": map[string]interface{}{
				"service.type": "NodePort",
				"service.port": 80,
			},
		},
	})
	assert.Nil(t, err)
}

// TestHelmTargetProviderInstall tests the Apply function of HelmTargetProvider
func TestHelmTargetProviderInstall(t *testing.T) {
	// To run this test case successfully, you shouldn't have a symphony Helm chart already deployed to your current Kubernetes context
	testSymphonyHelmVersion := os.Getenv("TEST_SYMPHONY_HELM_VERSION")
	if testSymphonyHelmVersion == "" {
		t.Skip("Skipping because TEST_SYMPHONY_HELM_VERSION environment variable is not set")
	}
	testCases := []struct {
		Name          string
		ChartRepo     string
		ExpectedError bool
	}{
		{Name: "install with wrong protocol", ChartRepo: fmt.Sprintf("wrongproto://%s", odooRepo), ExpectedError: true},
		{Name: "install with oci prefix", ChartRepo: fmt.Sprintf("oci://%s", odooRepo), ExpectedError: false},
		{Name: "install without oci prefix", ChartRepo: odooRepo, ExpectedError: false},
		// cleanup step is in TestHelmTargetProviderRemove
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := HelmTargetProviderConfig{InCluster: true}
			provider := HelmTargetProvider{}
			err := provider.Init(config)
			assert.Nil(t, err)
			component := model.ComponentSpec{
				Name: "odoo",
				Type: "helm.v3",
				Properties: map[string]interface{}{
					"chart": map[string]string{
						"repo":    tc.ChartRepo,
						"version": odooVersion,
					},
					"values": map[string]interface{}{
						"service.type": "NodePort",
						"service.port": 80,
					},
				},
			}
			deployment := model.DeploymentSpec{
				Solution: model.SolutionState{
					Spec: &model.SolutionSpec{
						Components: []model.ComponentSpec{component},
					},
				},
				Instance: model.InstanceState{
					ObjectMeta: model.ObjectMeta{
						Name: "test-instance",
					},
					Spec: &model.InstanceSpec{
						Scope: defaultTestScope,
					},
				},
			}
			step := model.DeploymentStep{
				Components: []model.ComponentStep{
					{
						Action:    model.ComponentUpdate,
						Component: component,
					},
				},
			}
			_, err = provider.Apply(context.Background(), deployment, step, false)
			assert.Equal(t, tc.ExpectedError, err != nil, "[TestCase: %s] failed. ExpectedError: %s", tc.Name, tc.ExpectedError)
		})
	}
}

// TestHelmTargetProviderGet tests the Get function of HelmTargetProvider
func TestHelmTargetProviderGet(t *testing.T) {
	// To run this test case successfully, you need to have a symphony Helm chart deployed to your current Kubernetes context
	testHelmChart := os.Getenv("TEST_HELM_CHART")
	if testHelmChart == "" {
		t.Skip("Skipping because TEST_HELM_CHART environment variable is not set")
	}

	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	components, err := provider.Get(context.Background(), model.DeploymentSpec{
		Instance: model.InstanceState{
			ObjectMeta: model.ObjectMeta{
				Name: "test-instance",
			},
			Spec: &model.InstanceSpec{
				Scope: defaultTestScope,
			},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{
					{
						Name: "odoo",
					},
				},
			},
		},
	}, []model.ComponentStep{
		{
			Action: model.ComponentUpdate,
			Component: model.ComponentSpec{
				Name: "odoo",
			},
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(components))
}

// TestHelmTargetProvider_NonOciChart tests the Apply function of HelmTargetProvider with no OCI registry
func TestHelmTargetProvider_NonOciChart(t *testing.T) {
	// To run this test case successfully, you shouldn't have a symphony Helm chart already deployed to your current Kubernetes context
	testSymphonyHelmVersion := os.Getenv("TEST_SYMPHONY_HELM_VERSION")
	if testSymphonyHelmVersion == "" {
		t.Skip("Skipping because TEST_SYMPHONY_HELM_VERSION environment variable is not set")
	}

	testCases := []struct {
		Name          string
		Chart         map[string]string
		Action        model.ComponentAction
		ExpectedError bool
	}{
		{
			Name: "repo URL not found ",
			Chart: map[string]string{
				"repo":    "https://not-found",
				"name":    "",
				"version": "",
			},
			Action:        model.ComponentUpdate,
			ExpectedError: true,
		},
		{
			Name: "chart not found in repo",
			Chart: map[string]string{
				"repo":    "https://project-akri.github.io/akri",
				"name":    "akri-not-found",
				"version": "",
			},
			Action:        model.ComponentUpdate,
			ExpectedError: true,
		},
		{
			Name: "version not found in repo",
			Chart: map[string]string{
				"repo":    "https://project-akri.github.io/akri",
				"name":    "akri",
				"version": "0.0.0",
			},
			Action:        model.ComponentUpdate,
			ExpectedError: true,
		},
		{
			Name: "update valid configuration without version",
			Chart: map[string]string{
				"repo":    "https://project-akri.github.io/akri",
				"name":    "akri",
				"version": "",
			},
			Action:        model.ComponentUpdate,
			ExpectedError: false,
		},
		{
			Name: "update valid configuration with version",
			Chart: map[string]string{
				"repo":    "https://project-akri.github.io/akri",
				"name":    "akri",
				"version": "0.12.9",
			},
			Action:        model.ComponentUpdate,
			ExpectedError: false,
		},
		{
			Name: "delete non-oci chart",
			Chart: map[string]string{
				"repo":    "https://project-akri.github.io/akri",
				"name":    "akri",
				"version": "0.12.9",
			},
			Action:        model.ComponentDelete,
			ExpectedError: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := HelmTargetProviderConfig{InCluster: true}
			provider := HelmTargetProvider{}
			err := provider.Init(config)
			assert.Nil(t, err)
			component := model.ComponentSpec{
				Name: "akri",
				Type: "helm.v3",
				Properties: map[string]interface{}{
					"chart": tc.Chart,
				},
			}
			deployment := model.DeploymentSpec{
				Solution: model.SolutionState{
					Spec: &model.SolutionSpec{
						Components: []model.ComponentSpec{component},
					},
				},
				Instance: model.InstanceState{
					ObjectMeta: model.ObjectMeta{
						Name: "test-instance-no-oci",
					},
					Spec: &model.InstanceSpec{
						Scope: defaultTestScope,
					},
				},
			}
			step := model.DeploymentStep{
				Components: []model.ComponentStep{
					{
						Action:    tc.Action,
						Component: component,
					},
				},
			}
			_, err = provider.Apply(context.Background(), deployment, step, false)
			assert.Equal(t, tc.ExpectedError, err != nil, "[chart %s]: %s failed. ExpectedError: %s", tc.Action, tc.Name, tc.ExpectedError)
		})
	}
}

func TestHelmTargetProviderInstallNginxIngress(t *testing.T) {
	// To run this test case successfully, you shouldn't have a symphony Helm chart already deployed to your current Kubernetes context
	testSymphonyHelmVersion := os.Getenv("TEST_SYMPHONY_HELM_VERSION")
	if testSymphonyHelmVersion == "" {
		t.Skip("Skipping because TEST_SYMPHONY_HELM_VERSION environment variable is not set")
	}

	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "ingress-nginx",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo":    "https://github.com/kubernetes/ingress-nginx/releases/download/helm-chart-4.7.1/ingress-nginx-4.7.1.tgz",
				"name":    "ingress-nginx",
				"version": "4.7.1",
			},
			"values": map[string]interface{}{
				"controller": map[string]interface{}{
					"replicaCount": 1,
					"nodeSelector": map[string]interface{}{
						"kubernetes.io/os": "linux",
					},
					"admissionWebhooks": map[string]interface{}{
						"patch": map[string]interface{}{
							"nodeSelector": map[string]interface{}{
								"kubernetes.io/os": "linux",
							},
						},
					},
					"service": map[string]interface{}{
						"annotations": map[string]interface{}{
							"service.beta.kubernetes.io/azure-load-balancer-health-probe-request-path": "/healthz",
						},
					},
				},
				"defaultBackend": map[string]interface{}{
					"nodeSelector": map[string]interface{}{
						"kubernetes.io/os": "linux",
					},
				},
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)

	time.Sleep(3 * time.Second)
	// cleanup
	step = model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentDelete,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)
}

// TestHelmTargetProviderInstallDirectDownload tests the Apply function of HelmTargetProvider with direct download
func TestHelmTargetProviderInstallDirectDownload(t *testing.T) {
	testGatekeeper := os.Getenv("TEST_HELM_GATEKEEPER")
	if testGatekeeper == "" {
		t.Skip("Skipping because TEST_HELM_GATEKEEPER environment variable is not set")
	}

	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "hello-world",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo": "https://github.com/helm/examples/releases/download/hello-world-0.1.0/hello-world-0.1.0.tgz",
				"name": "hello-world",
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			ObjectMeta: model.ObjectMeta{
				Name: "test-instance",
			},
			Spec: &model.InstanceSpec{
				Scope: defaultTestScope,
			},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)
}

// TestHelmTargetProviderRemove tests the Remove function of HelmTargetProvider
func TestHelmTargetProviderRemove(t *testing.T) {
	testSymphonyHelmVersion := os.Getenv("TEST_SYMPHONY_HELM_VERSION")
	if testSymphonyHelmVersion == "" {
		t.Skip("Skipping because TEST_SYMPHONY_HELM_VERSION environment variable is not set")
	}

	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "odoo",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo":    odooRepo,
				"version": odooVersion,
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			ObjectMeta: model.ObjectMeta{
				Name: "test-instance",
			},
			Spec: &model.InstanceSpec{
				Scope: defaultTestScope,
			},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentDelete,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)
}

// TestHelmTargetProviderGetAnotherCluster tests the Get function of HelmTargetProvider with another cluster
func TestHelmTargetProviderGetAnotherCluster(t *testing.T) {
	//to run this test successfully, you need to fix the cluster config below, and the target cluster shouldn't have symphony Helm chart installed
	//THIS CASE IS BROKERN
	testotherK8s := os.Getenv("TEST_HELM_OTHER_K8S")
	if testotherK8s == "" {
		t.Skip("Skipping because TEST_HELM_OTHER_K8S environment variable is not set")
	}
	config := HelmTargetProviderConfig{
		InCluster:  false,
		ConfigType: "bytes",
		ConfigData: `apiVersion: v1
 clusters:
 - cluster:
	 certificate-authority-data: ...
	 server: https://k12s-dns-6b5afdc5.hcp.westus3.azmk8s.io:443
 name: k12s
 contexts:
 - context:
	 cluster: k12s
	 user: clusterUser_symphony_k12s
 name: k12s
 current-context: k12s
 kind: Config
 preferences: {}
 users:
 - name: clusterUser_symphony_k12s
 user:
	 client-certificate-data: ...
	 client-key-data: ...
	 token: ...`,
	}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	components, err := provider.Get(context.Background(), model.DeploymentSpec{}, nil)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(components))
}

func TestHelmTargetProviderUpdateDelete(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "odoo",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo":    "registry-1.docker.io/bitnamicharts/odoo",
				"name":    "odoo",
				"version": "28.1.1",
			},
			"values": map[string]interface{}{
				"service.type": "NodePort",
				"service.port": 80,
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)

	step = model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentDelete,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)
}

func TestHelmTargetProviderWithNegativeTimeout(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "nginx",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]any{
				"repo":    "https://github.com/kubernetes/ingress-nginx/releases/download/helm-chart-4.7.1/ingress-nginx-4.7.1.tgz",
				"name":    "nginx",
				"wait":    true,
				"timeout": "-10s",
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	fmt.Printf("error timeout %v", err.Error())
	if !strings.Contains(err.Error(), "timeout can not be negative") {
		t.Errorf("expected error to contain 'timeout can not be negative', but got %s", err.Error())
	}
	assert.NotNil(t, err)
}

func TestHelmTargetProviderWithPositiveTimeout(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "brigade",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]any{
				"repo":    "https://brigadecore.github.io/charts",
				"name":    "brigade",
				"wait":    true,
				"timeout": "0.01s",
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected error to contain 'context deadline exceeded', but got %s", err.Error())
	}
	assert.NotNil(t, err)
}

func TestHelmTargetProviderWithInvalidTimeout(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "brigade",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]any{
				"repo":    "https://brigadecore.github.io/charts",
				"name":    "brigade",
				"wait":    true,
				"timeout": "20ssss",
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	if !strings.Contains(err.Error(), "time: unknown unit ") {
		t.Errorf("expected error to contain 'time: unknown unit', but got %s", err.Error())
	}
	assert.NotNil(t, err)
}

func TestHelmTargetProviderUpdateFailed(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	component := model.ComponentSpec{
		Name: "odoo",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo":    "abc/def",
				"name":    "odoo",
				"version": "0.1.1",
			},
			"values": map[string]interface{}{
				"service.type": "NodePort",
				"service.port": 80,
			},
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}
	_, err = provider.Apply(context.Background(), deployment, step, false)
	assert.NotNil(t, err)
}

func TestHelmTargetProviderGetEmpty(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	_, err = provider.Get(context.Background(), model.DeploymentSpec{
		Instance: model.InstanceState{
			Spec: &model.InstanceSpec{},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{
					{
						Name: "odoo",
					},
				},
			},
		},
	}, []model.ComponentStep{
		{
			Action: model.ComponentUpdate,
			Component: model.ComponentSpec{
				Name: "odoo",
			},
		},
	})
	assert.Nil(t, err)
}

func TestDownloadFile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	err := downloadFile(ts.URL, "test")
	assert.Nil(t, err)
	_ = os.Remove("test")
}

func TestGetActionConfig(t *testing.T) {
	config := &rest.Config{
		Host:        "host",
		BearerToken: "token",
	}
	_, err := getActionConfig(context.Background(), "default", config)
	assert.Nil(t, err)
}

// TestConformanceSuite tests the HelmTargetProvider for conformance
func TestConformanceSuite(t *testing.T) {
	provider := &HelmTargetProvider{}
	err := provider.Init(HelmTargetProviderConfig{InCluster: true})
	assert.Nil(t, err)
	conformance.ConformanceSuite(t, provider)
}

func TestPropChange(t *testing.T) {
	type PropertyChangeCase struct {
		Name    string
		OldProp map[string]interface{}
		NewProp map[string]interface{}
		Changed bool
	}
	var cases = []PropertyChangeCase{
		{"Both nil", nil, nil, false},
		{"Old empty New nil", map[string]interface{}{}, nil, false},
		{"Old nil New empty", nil, map[string]interface{}{}, false},
		{"No change", map[string]interface{}{"a": "b"}, map[string]interface{}{"a": "b"}, false},
		{"Balue changed", map[string]interface{}{"a": "b"}, map[string]interface{}{"a": "c"}, true},
		{"New property added", map[string]interface{}{"a": "b"}, map[string]interface{}{"a": "b", "c": "d"}, true},
		{"Property removed", map[string]interface{}{"a": "b", "c": "d"}, map[string]interface{}{"a": "b"}, true},
		{"Property order changed", map[string]interface{}{"a": "b", "c": "d"}, map[string]interface{}{"c": "d", "a": "b"}, false},
	}

	for _, c := range cases {
		assert.Equal(t, c.Changed, propChange(c.OldProp, c.NewProp), c.Name)
	}
}

func TestConfigureInstallClient(t *testing.T) {
	ctx := context.Background()
	actionConfig := &action.Configuration{}
	postRenderer := postrender.PostRenderer(nil)

	tests := []struct {
		name           string
		releaseName    string
		componentProps HelmChartProperty
		deployment     model.DeploymentSpec
		expectedName   string
		expectedError  bool
	}{
		{
			name:        "Custom release name provided",
			releaseName: "custom-release",
			componentProps: HelmChartProperty{
				Wait:    true,
				Timeout: "30s",
			},
			deployment: model.DeploymentSpec{
				Instance: model.InstanceState{
					Spec: &model.InstanceSpec{
						Scope: "test-namespace",
					},
				},
			},
			expectedName:  "custom-release",
			expectedError: false,
		},
		{
			name:        "Invalid timeout format",
			releaseName: "invalid-timeout-release",
			componentProps: HelmChartProperty{
				Wait:    true,
				Timeout: "invalid-timeout",
			},
			deployment: model.DeploymentSpec{
				Instance: model.InstanceState{
					Spec: &model.InstanceSpec{
						Scope: "test-namespace",
					},
				},
			},
			expectedName:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installClient, err := configureInstallClient(ctx, tt.releaseName, &tt.componentProps, &tt.deployment, actionConfig, postRenderer)
			if tt.expectedError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedName, installClient.ReleaseName)
				assert.Equal(t, tt.deployment.Instance.Spec.Scope, installClient.Namespace)
				if tt.componentProps.Timeout != "" {
					duration, _ := time.ParseDuration(tt.componentProps.Timeout)
					assert.Equal(t, duration, installClient.Timeout)
				}
			}
		})
	}
}

func TestHelmTargetProviderApplyWithCustomReleaseName(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED environment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)

	customReleaseName := "custom-release-name"
	component := model.ComponentSpec{
		Name: "kashti",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo": "https://brigadecore.github.io/charts",
				"name": "kashti",
			},
			"releaseName": customReleaseName,
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			ObjectMeta: model.ObjectMeta{
				Name: "test-instance",
			},
			Spec: &model.InstanceSpec{
				Scope: defaultTestScope,
			},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	step := model.DeploymentStep{
		Components: []model.ComponentStep{
			{
				Action:    model.ComponentUpdate,
				Component: component,
			},
		},
	}

	// Apply the Helm chart with the custom release name
	results, err := provider.Apply(context.Background(), deployment, step, false)
	assert.Nil(t, err)
	assert.Equal(t, v1alpha2.Updated, results[component.Name].Status)
	assert.Contains(t, results[component.Name].Message, customReleaseName)

	// Verify the release name using Helm client
	settings := cli.New()
	actionConfig := &action.Configuration{}
	err = actionConfig.Init(settings.RESTClientGetter(), defaultTestScope, "secrets", func(format string, v ...interface{}) {})
	assert.Nil(t, err)

	listClient := action.NewList(actionConfig)
	listClient.AllNamespaces = true
	releases, err := listClient.Run()
	assert.Nil(t, err)

	found := false
	for _, release := range releases {
		if release.Name == customReleaseName {
			fmt.Printf("Found release with custom name: %s\n", release.Name)
			found = true
			break
		}
	}
	assert.True(t, found, "Custom release name not found in Helm releases")
}

func TestHelmTargetProviderGetWithCustomReleaseName(t *testing.T) {
	testEnabled := os.Getenv("TEST_MINIKUBE_ENABLED")
	if testEnabled == "" {
		t.Skip("Skipping because TEST_MINIKUBE_ENABLED enviornment variable is not set")
	}
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)

	customReleaseName := "custom-release-name"
	component := model.ComponentSpec{
		Name: "kashti",
		Type: "helm.v3",
		Properties: map[string]interface{}{
			"chart": map[string]string{
				"repo": "https://brigadecore.github.io/charts",
				"name": "kashti",
			},
			"releaseName": customReleaseName,
		},
	}
	deployment := model.DeploymentSpec{
		Instance: model.InstanceState{
			ObjectMeta: model.ObjectMeta{
				Name: "test-instance",
			},
			Spec: &model.InstanceSpec{
				Scope: defaultTestScope,
			},
		},
		Solution: model.SolutionState{
			Spec: &model.SolutionSpec{
				Components: []model.ComponentSpec{component},
			},
		},
	}
	references := []model.ComponentStep{
		{
			Action:    model.ComponentUpdate,
			Component: component,
		},
	}

	// Get the Helm release with the custom release name
	components, err := provider.Get(context.Background(), deployment, references)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(components))
	assert.Equal(t, customReleaseName, components[0].Properties["releaseName"])
}

func TestHelmTargetProviderPullChartUnauthorizedError(t *testing.T) {
	config := HelmTargetProviderConfig{InCluster: true}
	provider := HelmTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)

	chart := &HelmChartProperty{
		Repo:    "oci://symphonyprivate.azurecr.io/helm/simple-chart-secure",
		Version: "0.1.0",
	}

	_, err = pullOCIChart(context.Background(), chart.Repo, chart.Version)
	assert.NotNil(t, err)
	assert.True(t, isUnauthorized(err))
}
