/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package federation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	federationv1 "gopls-workspace/apis/federation/v1"
)

// SiteReconciler reconciles a Site object
type SiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=federation.symphony,resources=sites,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=federation.symphony,resources=sites/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=federation.symphony,resources=sites/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Device object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// We need to re-able recoverPanic once the behavior is tested #691
	recoverPanic := false
	return ctrl.NewControllerManagedBy(mgr).
		Named("Site").
		WithOptions((controller.Options{RecoverPanic: &recoverPanic})).
		For(&federationv1.Site{}).
		Complete(r)
}
