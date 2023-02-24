package http

import (
	"context"
	"os"
	"testing"

	"github.com/azure/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/stretchr/testify/assert"
)

func TestHttpTargetProviderConfigFromMapNil(t *testing.T) {
	_, err := HttpTargetProviderConfigFromMap(nil)
	assert.Nil(t, err)
}
func TestHttpTargetProviderConfigFromMapEmpty(t *testing.T) {
	_, err := HttpTargetProviderConfigFromMap(map[string]string{})
	assert.Nil(t, err)
}

func TestHttpTargetProviderApply(t *testing.T) {
	testLogicApps := os.Getenv("TEST_LOGIC_APPS")
	if testLogicApps == "" {
		t.Skip("Skipping because TEST_LOGIC_APPS enviornment variable is not set")
	}
	config := HttpTargetProviderConfig{
		Name: "qa-target",
	}
	provider := HttpTargetProvider{}
	err := provider.Init(config)
	assert.Nil(t, err)
	err = provider.Apply(context.Background(), model.DeploymentSpec{
		Solution: model.SolutionSpec{
			Components: []model.ComponentSpec{
				{
					Name: "http-component",
					Properties: map[string]string{
						"http.url":    "https://manual-approval.azurewebsites.net:443/api/approval/triggers/manual/invoke?api-version=2022-05-01&sp=%2Ftriggers%2Fmanual%2Frun&sv=1.0&sig=<sig>",
						"http.method": "POST",
						"http.body":   `{"solution":"$solution()", "instance": "$instance()", "target": "$target()", "id": "$instance()-$solution()-$target()"}`,
					},
				},
			},
		},
		Assignments: map[string]string{
			"target-1": "{http-component}",
		},
		Targets: map[string]model.TargetSpec{
			"target-1": {
				Topologies: []model.TopologySpec{
					{
						Bindings: []model.BindingSpec{
							{
								Role:     "instance",
								Provider: "doesn't-matter",
								Config:   map[string]string{},
							},
						},
					},
				},
			},
		},
	})
	assert.Nil(t, err)
}