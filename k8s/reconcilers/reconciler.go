/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package reconcilers

import (
	"context"
	apiV1 "gopls-workspace/apis/model/v1"
	"gopls-workspace/controllers/metrics"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type (
	Reconciler interface {
		AttemptUpdate(ctx context.Context, object Reconcilable, isRemoval bool, logger logr.Logger, operationStartTimeKey string, operationName string) (metrics.OperationStatus, reconcile.Result, error)
		PollingResult(ctx context.Context, object Reconcilable, isRemoval bool, logger logr.Logger, operationStartTimeKey string, operationName string) (metrics.OperationStatus, reconcile.Result, error)
	}
	Reconcilable interface {
		client.Object
		GetStatus() apiV1.DeployableStatusV2
		SetStatus(apiV1.DeployableStatusV2)
		GetReconciliationPolicy() *apiV1.ReconciliationPolicySpec
	}
)
