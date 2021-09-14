// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// Package operator holds the churro operator logic
package operator

import (
	"github.com/churrodata/churro/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	jobOwnerKey = ".metadata.controller"
)

func (r PipelineReconciler) processRoles(pipeline v1alpha1.Pipeline) error {
	// get referenced rbac objects
	var childRoles rbacv1.RoleList
	err := r.List(r.Ctx, &childRoles, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child roles")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a role/churro
	needChurroRole := true
	// make sure we have a role/cockroachdb
	needCockroachRole := true
	for i := 0; i < len(childRoles.Items); i++ {
		r := childRoles.Items[i]
		if r.Name == "churro" {
			needChurroRole = false
		} else if r.Name == "cockroachdb" {
			needCockroachRole = false
		}
	}

	// create any expected rbac objects, set owner reference to this pipeline
	rolesToCreate := make([]rbacv1.Role, 0)
	if needChurroRole {
		churroRole := rbacv1.Role{}
		churroRole.Name = "churro"
		churroRole.Namespace = pipeline.ObjectMeta.Namespace
		/**
		  rules:
		  - apiGroups:
		    - ""
		    resources:
		    - pods
		    - pods/log
		    - services
		    - secrets
		    verbs:
		    - create
		    - get
		    - list
		    - delete
		  - apiGroups:
		    - "churro.project.io"
		    resources:
		    - pipelines
		    verbs:
		    - get
		    - list
		    - update
		*/
		churroRole.Rules = make([]rbacv1.PolicyRule, 0)
		policyRule := rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"create", "get", "list", "delete"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"pods", "pods/log", "services", "secrets"}
		churroRole.Rules = append(churroRole.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"get", "list", "update"}
		policyRule.APIGroups = []string{"churro.project.io"}
		policyRule.Resources = []string{"pipelines"}
		churroRole.Rules = append(churroRole.Rules, policyRule)

		rolesToCreate = append(rolesToCreate, churroRole)
	}
	if needCockroachRole {
		cockroachRole := rbacv1.Role{}
		cockroachRole.Name = "cockroachdb"
		cockroachRole.Namespace = pipeline.ObjectMeta.Namespace
		/**
		  rules:
		  - apiGroups:
		    - ""
		    resources:
		    - secrets
		    verbs:
		    - get
		*/
		cockroachRole.Rules = make([]rbacv1.PolicyRule, 0)
		policyRule := rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"get"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"secrets"}
		cockroachRole.Rules = append(cockroachRole.Rules, policyRule)

		rolesToCreate = append(rolesToCreate, cockroachRole)
	}

	for _, role := range rolesToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &role, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &role); err != nil {
			r.Log.Error(err, "unable to create rbacObject for pipeline", "rbac", role)
			return err
		}
		r.Log.Info("created rbac for pipeline ")
	}

	return nil
}

func (r PipelineReconciler) processServiceAccounts(pipeline v1alpha1.Pipeline) error {
	var childSAs v1.ServiceAccountList
	err := r.List(r.Ctx, &childSAs, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child serviceaccounts")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a sa/churro
	needChurroSA := true
	// make sure we have a sa/cockroachdb
	needCockroachSA := true
	for i := 0; i < len(childSAs.Items); i++ {
		r := childSAs.Items[i]
		if r.Name == "churro" {
			needChurroSA = false
		} else if r.Name == "cockroachdb" {
			needCockroachSA = false
		}
	}

	// create any expected rbac objects, set owner reference to this pipeline
	accountsToCreate := make([]v1.ServiceAccount, 0)
	if needChurroSA {
		churroSA := v1.ServiceAccount{}
		churroSA.Name = "churro"
		churroSA.Namespace = pipeline.ObjectMeta.Namespace
		/**
		apiVersion: v1
		kind: ServiceAccount
		metadata:
		  name: churro
		  labels:
		    app: churro
		*/
		accountsToCreate = append(accountsToCreate, churroSA)
	}
	if needCockroachSA {
		cockroachSA := v1.ServiceAccount{}
		cockroachSA.Name = "cockroachdb"
		cockroachSA.Namespace = pipeline.ObjectMeta.Namespace
		/**
		apiVersion: v1
		kind: ServiceAccount
		metadata:
		  name: cockroachdb
		  labels:
		    app: cockroachdb

		*/
		accountsToCreate = append(accountsToCreate, cockroachSA)
	}

	for _, acct := range accountsToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &acct, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &acct); err != nil {
			r.Log.Error(err, "unable to create service account for pipeline", "rbac", acct)
			return err
		}
		r.Log.Info("created service account for pipeline ")
	}

	return nil
}

func (r PipelineReconciler) processRoleBindings(pipeline v1alpha1.Pipeline) error {
	// get referenced rbac objects
	var childRoleBindings rbacv1.RoleBindingList
	if err := r.List(r.Ctx, &childRoleBindings, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name}); err != nil {
		r.Log.Error(err, "unable to list child rolebindings")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a rolebinding/churro
	needChurroRoleBinding := true
	// make sure we have a rolebinding/cockroachdb
	needCockroachRoleBinding := true
	for i := 0; i < len(childRoleBindings.Items); i++ {
		r := childRoleBindings.Items[i]
		if r.Name == "churro" {
			needChurroRoleBinding = false
		} else if r.Name == "cockroachdb" {
			needCockroachRoleBinding = false
		}
	}

	// create any expected rbac objects, set owner reference to this pipeline
	roleBindingsToCreate := make([]rbacv1.RoleBinding, 0)
	if needChurroRoleBinding {
		churroRoleBinding := rbacv1.RoleBinding{}
		churroRoleBinding.Name = "churro"
		churroRoleBinding.Namespace = pipeline.ObjectMeta.Namespace
		/**
		kind: RoleBinding
		apiVersion: rbac.authorization.k8s.io/v1
		metadata:
		  name: churro
		subjects:
		- kind: ServiceAccount
		  name: churro
		roleRef:
		  kind: Role
		  name: churro
		  apiGroup: rbac.authorization.k8s.io
		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "churro"
		subject.Namespace = pipeline.ObjectMeta.Namespace
		churroRoleBinding.Subjects = []rbacv1.Subject{subject}
		churroRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "churro"}
		churroRoleBinding.ObjectMeta.Name = subject.Name
		churroRoleBinding.ObjectMeta.Labels = map[string]string{"app": subject.Name}
		roleBindingsToCreate = append(roleBindingsToCreate, churroRoleBinding)
	}
	if needCockroachRoleBinding {
		cockroachRoleBinding := rbacv1.RoleBinding{}
		cockroachRoleBinding.Name = "cockroachdb"
		cockroachRoleBinding.Namespace = pipeline.ObjectMeta.Namespace
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: RoleBinding
		metadata:
		  name: cockroachdb
		  labels:
		    app: cockroachdb
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: Role
		  name: cockroachdb
		subjects:
		- kind: ServiceAccount
		  name: cockroachdb
		  namespace: default
		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "cockroachdb"
		subject.Namespace = pipeline.ObjectMeta.Namespace
		cockroachRoleBinding.Subjects = []rbacv1.Subject{subject}
		cockroachRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "cockroachdb"}

		cockroachRoleBinding.ObjectMeta.Name = subject.Name
		cockroachRoleBinding.ObjectMeta.Labels = map[string]string{"app": subject.Name}
		roleBindingsToCreate = append(roleBindingsToCreate, cockroachRoleBinding)
	}

	for _, roleBinding := range roleBindingsToCreate {
		if err := ctrl.SetControllerReference(&pipeline, &roleBinding, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &roleBinding); err != nil {
			r.Log.Error(err, "unable to create roleBinding for pipeline", "rbac", roleBinding)
			return err
		}
		r.Log.Info("created rolebinding for pipeline ")
	}

	return nil
}
