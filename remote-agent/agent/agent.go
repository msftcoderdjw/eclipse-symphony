/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package agent

import (
	"context"
	"encoding/json"
	"errors"

	tgt "github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/providers/target"
	utils "github.com/eclipse-symphony/symphony/remote-agent/common"
)

type Agent struct {
	Providers map[string]tgt.ITargetProvider
}

func (s Agent) Handle(req []byte, ctx context.Context) utils.AsyncResult {
	request := utils.AgentRequest{}
	err := json.Unmarshal(req, &request)
	if err != nil {
		return utils.AsyncResult{OperationID: request.OperationID, Error: err}
	}

	body := new([]byte)

	provider := s.Providers[request.Provider]
	if provider == nil {
		return utils.AsyncResult{OperationID: request.OperationID, Error: errors.New("Provider not found")}
	}

	switch request.Action {
	case "get":
		var getRequest utils.ProviderGetRequest
		if err := json.Unmarshal(req, &getRequest); err != nil {
			return utils.AsyncResult{OperationID: request.OperationID, Error: err}
		}
		specs, err := provider.Get(ctx, getRequest.Deployment, getRequest.References)
		*body, err = json.Marshal(specs)
		return utils.AsyncResult{OperationID: request.OperationID, Body: *body, Error: err}

	case "apply":
		var applyRequest utils.ProviderApplyRequest
		if err := json.Unmarshal(req, &applyRequest); err != nil {
			return utils.AsyncResult{OperationID: request.OperationID, Error: err}
		}
		specs, err := provider.Apply(ctx, applyRequest.Deployment, applyRequest.Step, applyRequest.Deployment.IsDryRun)
		*body, err = json.Marshal(specs)
		return utils.AsyncResult{OperationID: request.OperationID, Body: *body, Error: err}

	case "getValidationRule":
		var getValidationRuleRequest utils.ProviderGetValidationRuleRequest
		if err := json.Unmarshal(req, &getValidationRuleRequest); err != nil {
			return utils.AsyncResult{OperationID: request.OperationID, Error: err}
		}
		rule := provider.GetValidationRule(ctx)
		*body, err = json.Marshal(rule)
		return utils.AsyncResult{OperationID: request.OperationID, Body: *body, Error: err}
	default:
		return utils.AsyncResult{OperationID: request.OperationID, Error: errors.New("Action not found")}
	}
}