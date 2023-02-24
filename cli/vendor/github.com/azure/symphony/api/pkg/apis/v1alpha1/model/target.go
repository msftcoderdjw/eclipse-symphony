/*
Copyright 2022 The COA Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import "errors"

type TargetSpec struct {
	DisplayName   string            `json:"displayName,omitempty"`
	Scope         string            `json:"scope,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Properties    map[string]string `json:"properties,omitempty"`
	Components    []ComponentSpec   `json:"components,omitempty"`
	Constraints   []ConstraintSpec  `json:"constraints,omitempty"`
	Topologies    []TopologySpec    `json:"topologies,omitempty"`
	ForceRedeploy bool              `json:"forceRedeploy,omitempty"`
}

func (c TargetSpec) DeepEquals(other IDeepEquals) (bool, error) {
	var otherC TargetSpec
	var ok bool
	if otherC, ok = other.(TargetSpec); !ok {
		return false, errors.New("parameter is not a TargetSpec type")
	}
	if c.DisplayName != otherC.DisplayName {
		return false, nil
	}
	if c.Scope != otherC.Scope {
		return false, nil
	}
	if !StringMapsEqual(c.Metadata, otherC.Metadata, nil) {
		return false, nil
	}
	if !StringMapsEqual(c.Properties, otherC.Properties, nil) {
		return false, nil
	}
	if !SlicesEqual(c.Components, otherC.Components) {
		return false, nil
	}
	if !SlicesEqual(c.Constraints, otherC.Constraints) {
		return false, nil
	}
	if !SlicesEqual(c.Topologies, otherC.Topologies) {
		return false, nil
	}
	if c.ForceRedeploy != otherC.ForceRedeploy {
		return false, nil
	}
	return true, nil
}