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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var solutionlog = logf.Log.WithName("solution-resource")
var mySolutionClient client.Client
var mySolutionReaderClient client.Reader

func (r *Solution) SetupWebhookWithManager(mgr ctrl.Manager) error {
	mySolutionClient = mgr.GetClient()
	mySolutionReaderClient = mgr.GetAPIReader()
	mgr.GetFieldIndexer().IndexField(context.Background(), &Solution{}, ".spec.displayName", func(rawObj client.Object) []string {
		solution := rawObj.(*Solution)
		return []string{solution.Spec.DisplayName}
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

	if r.Spec.RootResource != "" {
		var solutionContainer SolutionContainer
		err := mySolutionClient.Get(context.Background(), client.ObjectKey{Name: r.Spec.RootResource, Namespace: r.Namespace}, &solutionContainer)
		if err != nil {
			solutionlog.Error(err, "failed to get solution container", "name", r.Spec.RootResource)
		} else {
			ownerReference := metav1.OwnerReference{
				APIVersion: GroupVersion.String(),
				Kind:       "SolutionContainer",
				Name:       solutionContainer.Name,
				UID:        solutionContainer.UID,
			}

			if !configutils.CheckOwnerReferenceAlreadySet(r.OwnerReferences, ownerReference) {
				r.OwnerReferences = append(r.OwnerReferences, ownerReference)
			}

			if r.Labels == nil {
				r.Labels = make(map[string]string)
			}
			r.Labels["rootResource"] = r.Spec.RootResource
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.

//+kubebuilder:webhook:path=/validate-solution-symphony-v1-solution,mutating=false,failurePolicy=fail,sideEffects=None,groups=solution.symphony,resources=solutions,verbs=create;update,versions=v1,name=vsolution.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Solution{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Solution) ValidateCreate() (admission.Warnings, error) {
	solutionlog.Info("validate create", "name", r.Name)

	return nil, r.validateCreateSolution()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Solution) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	solutionlog.Info("validate update", "name", r.Name)

	return nil, r.validateUpdateSolution()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Solution) ValidateDelete() (admission.Warnings, error) {
	solutionlog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func (r *Solution) validateCreateSolution() error {
	var allErrs field.ErrorList

	if err := r.validateUniqueNameOnCreate(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateNameOnCreate(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateRootResource(); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: "solution.symphony", Kind: "Solution"}, r.Name, allErrs)
}

func (r *Solution) validateUpdateSolution() error {
	var solutions SolutionList
	err := mySolutionClient.List(context.Background(), &solutions, client.InNamespace(r.Namespace), client.MatchingFields{".spec.displayName": r.Spec.DisplayName}, client.Limit(2))
	if err != nil {
		return apierrors.NewInternalError(err)
	}
	if !(len(solutions.Items) == 0 || len(solutions.Items) == 1 && solutions.Items[0].ObjectMeta.Name == r.ObjectMeta.Name) {
		return apierrors.NewBadRequest(fmt.Sprintf("solution display name '%s' is already taken", r.Spec.DisplayName))
	}
	return nil
}

func (r *Solution) validateUniqueNameOnCreate() *field.Error {
	var solutions SolutionList
	err := mySolutionClient.List(context.Background(), &solutions, client.InNamespace(r.Namespace), client.MatchingFields{".spec.displayName": r.Spec.DisplayName}, client.Limit(1))
	if err != nil {
		return field.InternalError(&field.Path{}, err)
	}

	if len(solutions.Items) != 0 {
		return field.Invalid(field.NewPath("spec").Child("displayName"), r.Spec.DisplayName, "solution display name is already taken")
	}

	return nil
}

func (r *Solution) validateNameOnCreate() *field.Error {
	return configutils.ValidateObjectName(r.ObjectMeta.Name, r.Spec.RootResource)
}

func (r *Solution) validateRootResource() *field.Error {
	var solutionContainer SolutionContainer
	err := mySolutionClient.Get(context.Background(), client.ObjectKey{Name: r.Spec.RootResource, Namespace: r.Namespace}, &solutionContainer)
	if err != nil {
		return field.Invalid(field.NewPath("spec").Child("rootResource"), r.Spec.RootResource, "rootResource must be a valid solution container")
	}

	if len(r.ObjectMeta.OwnerReferences) == 0 {
		return field.Invalid(field.NewPath("metadata").Child("ownerReference"), len(r.ObjectMeta.OwnerReferences), "ownerReference must be set")
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
