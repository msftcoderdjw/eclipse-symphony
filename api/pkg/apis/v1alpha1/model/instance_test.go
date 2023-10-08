/*

	MIT License

	Copyright (c) Microsoft Corporation.

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE

*/

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceDeepEquals(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.True(t, res)
}

func TestInstanceDeepEqualsOneEmpty(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(nil)
	assert.Errorf(t, err, "parameter is not a InstanceSpec type")
	assert.False(t, res)
}

func TestInstanceDeepEqualsNameNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName1",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceDeepEqualsDisplayNameNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName1",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceDeepEqualsScopeNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default1",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceDeepEqualsTargetNameNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName1",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		}}

	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceDeepEqualsTopologiestNameNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName1",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceEqualsPipelineNameNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName1",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceEqualsArgumentsKeysNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo1": {"foo": "bar"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestInstanceEqualsArgumentsValuesNotMatch(t *testing.T) {
	Instance := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo": "bar"},
		},
	}
	other := InstanceSpec{
		Name:        "InstanceName",
		DisplayName: "InstanceDisplayName",
		Scope:       "Default",
		Metadata: map[string]string{
			"foo": "bar",
		},
		Solution: "SolutionName",
		Target: TargetSelector{
			Name: "TargetName",
		},
		Topologies: []TopologySpec{{
			Device: "DeviceName",
		}},
		Pipelines: []PipelineSpec{{
			Name: "PipelineName",
		}},
		Arguments: map[string]map[string]string{
			"foo": {"foo1": "bar1"},
		},
	}
	res, err := Instance.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestTargetSelectorDeepEqualsOneEmpty(t *testing.T) {
	Target := TargetSelector{
		Name: "TargetName",
	}
	res, err := Target.DeepEquals(nil)
	assert.Errorf(t, err, "parameter is not a TargetSelector type")
	assert.False(t, res)
}

func TestTargetSelectorDeepEqualsSelectorNotMatch(t *testing.T) {
	Target := TargetSelector{
		Name: "TargetName",
		Selector: map[string]string{
			"foo": "bar",
		},
	}
	other := TargetSelector{
		Name: "TargetName",
		Selector: map[string]string{
			"foo1": "bar1",
		},
	}
	res, err := Target.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestPipelineSpecDeepEqualsOneEmpty(t *testing.T) {
	Pipeline := PipelineSpec{
		Name: "PipelineName",
	}
	res, err := Pipeline.DeepEquals(nil)
	assert.Errorf(t, err, "parameter is not a PipelineSpec type")
	assert.False(t, res)
}

func TestPipelineSpecDeepEqualsSkillNotMatch(t *testing.T) {
	Pipeline := PipelineSpec{
		Name:  "PipelineName",
		Skill: "skill",
		Parameters: map[string]string{
			"foo": "bar",
		},
	}
	other := PipelineSpec{
		Name:  "PipelineName",
		Skill: "skill1",
		Parameters: map[string]string{
			"foo": "bar",
		},
	}
	res, err := Pipeline.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestPipelineSpecDeepEqualsParametersNotMatch(t *testing.T) {
	Pipeline := PipelineSpec{
		Name:  "PipelineName",
		Skill: "skill",
		Parameters: map[string]string{
			"foo": "bar",
		},
	}
	other := PipelineSpec{
		Name:  "PipelineName",
		Skill: "skill",
		Parameters: map[string]string{
			"foo1": "bar1",
		},
	}
	res, err := Pipeline.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestTopologySpecDeepEqualsOneEmpty(t *testing.T) {
	Topology := TopologySpec{
		Device: "DeviceName",
	}
	res, err := Topology.DeepEquals(nil)
	assert.Errorf(t, err, "parameter is not a TopologySpec type")
	assert.False(t, res)
}

func TestTopologySpecDeepEqualsSelectorNotMatch(t *testing.T) {
	Topology := TopologySpec{
		Device: "DeviceName",
		Selector: map[string]string{
			"foo": "bar",
		},
	}
	other := TopologySpec{
		Device: "DeviceName",
		Selector: map[string]string{
			"foo1": "bar1",
		},
	}
	res, err := Topology.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}

func TestTopologySpecDeepEqualsBindingsNotMatch(t *testing.T) {
	Topology := TopologySpec{
		Device: "DeviceName",
		Selector: map[string]string{
			"foo": "bar",
		},
		Bindings: []BindingSpec{{
			Role: "RoleName",
		},
		},
	}
	other := TopologySpec{
		Device: "DeviceName",
		Selector: map[string]string{
			"foo": "bar",
		},
		Bindings: []BindingSpec{{
			Role: "RoleName1",
		},
		},
	}
	res, err := Topology.DeepEquals(other)
	assert.Nil(t, err)
	assert.False(t, res)
}