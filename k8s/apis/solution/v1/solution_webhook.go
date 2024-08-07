/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package v1

import (
	"context"
	"fmt"
	commoncontainer "gopls-workspace/apis/model/v1"
	"gopls-workspace/configutils"

	configv1 "gopls-workspace/apis/config/v1"

	observ_utils "github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var solutionlog = logf.Log.WithName("solution-resource")
var mySolutionClient client.Client

func (r *Solution) SetupWebhookWithManager(mgr ctrl.Manager) error {
	mySolutionClient = mgr.GetClient()
	mgr.GetFieldIndexer().IndexField(context.Background(), &Solution{}, ".spec.displayName", func(rawObj client.Object) []string {
		target := rawObj.(*Solution)
		return []string{target.Spec.DisplayName}
	})
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-solution-symphony-v1-solution,mutating=true,failurePolicy=fail,sideEffects=None,groups=solution.symphony,resources=solutions,verbs=create;update,versions=v1,name=msolution.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Solution{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Solution) Default() {
	solutionlog.Info("default", "name", r.Name)

	if r.Spec.DisplayName == "" {
		r.Spec.DisplayName = r.ObjectMeta.Name
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

//+kubebuilder:webhook:path=/validate-solution-symphony-v1-solution,mutating=false,failurePolicy=fail,sideEffects=None,groups=solution.symphony,resources=solutions,verbs=create;update,versions=v1,name=vsolution.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Solution{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Solution) ValidateCreate() (admission.Warnings, error) {
	solutionlog.Info("validate create", "name", r.Name)

	resourceK8SId := r.GetNamespace() + "/" + r.GetName()
	operationName := fmt.Sprintf("%s/%s", constants.SolutionOperationNamePrefix, constants.ActivityOperation_Write)
	ctx := configutils.PopulateActivityAndDiagnosticsContextFromAnnotations(resourceK8SId, r.Annotations, constants.ActivityCategory_Activity, operationName, context.TODO(), solutionlog)

	observ_utils.EmitUserAuditsLogs(ctx, "Activation %s is being created on namespace %s", r.Name, r.Namespace)

	return nil, r.validateCreateSolution()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Solution) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	solutionlog.Info("validate update", "name", r.Name)

	resourceK8SId := r.GetNamespace() + "/" + r.GetName()
	operationName := fmt.Sprintf("%s/%s", constants.SolutionOperationNamePrefix, constants.ActivityOperation_Write)
	ctx := configutils.PopulateActivityAndDiagnosticsContextFromAnnotations(resourceK8SId, r.Annotations, constants.ActivityCategory_Activity, operationName, context.TODO(), solutionlog)

	observ_utils.EmitUserAuditsLogs(ctx, "Activation %s is being updated on namespace %s", r.Name, r.Namespace)

	return nil, r.validateUpdateSolution()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Solution) ValidateDelete() (admission.Warnings, error) {
	solutionlog.Info("validate delete", "name", r.Name)

	resourceK8SId := r.GetNamespace() + "/" + r.GetName()
	operationName := fmt.Sprintf("%s/%s", constants.SolutionOperationNamePrefix, constants.ActivityOperation_Delete)
	ctx := configutils.PopulateActivityAndDiagnosticsContextFromAnnotations(resourceK8SId, r.Annotations, constants.ActivityCategory_Activity, operationName, context.TODO(), solutionlog)

	observ_utils.EmitUserAuditsLogs(ctx, "Activation %s is being deleted on namespace %s", r.Name, r.Namespace)

	return nil, nil
}

func (r *Solution) validateCreateSolution() error {
	var solutions SolutionList
	mySolutionClient.List(context.Background(), &solutions, client.InNamespace(r.Namespace), client.MatchingFields{".spec.displayName": r.Spec.DisplayName})
	if len(solutions.Items) != 0 {
		return apierrors.NewBadRequest(fmt.Sprintf("solution display name '%s' is already taken", r.Spec.DisplayName))
	}
	return nil
}

func (r *Solution) validateUpdateSolution() error {
	var solutions SolutionList
	err := mySolutionClient.List(context.Background(), &solutions, client.InNamespace(r.Namespace), client.MatchingFields{".spec.displayName": r.Spec.DisplayName})
	if err != nil {
		return apierrors.NewInternalError(err)
	}
	if !(len(solutions.Items) == 0 || len(solutions.Items) == 1 && solutions.Items[0].ObjectMeta.Name == r.ObjectMeta.Name) {
		return apierrors.NewBadRequest(fmt.Sprintf("solution display name '%s' is already taken", r.Spec.DisplayName))
	}
	return nil
}

func (r *SolutionContainer) Default() {
	commoncontainer.DefaultImpl(solutionlog, r)
}

func (r *SolutionContainer) ValidateCreate() (admission.Warnings, error) {
	return commoncontainer.ValidateCreateImpl(solutionlog, r)
}
func (r *SolutionContainer) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return commoncontainer.ValidateUpdateImpl(solutionlog, r, old)
}

func (r *SolutionContainer) ValidateDelete() (admission.Warnings, error) {
	solutionlog.Info("validate delete solution container", "name", r.Name)
	getSubResourceNums := func() (int, error) {
		var solutionList SolutionList
		err := mySolutionReaderClient.List(context.Background(), &solutionList, client.InNamespace(r.Namespace), client.MatchingLabels{"rootResource": r.Name}, client.Limit(1))
		if err != nil {
			return 0, err
		} else {
			return len(solutionList.Items), nil
		}
	}
	return commoncontainer.ValidateDeleteImpl(solutionlog, r, getSubResourceNums)
}
