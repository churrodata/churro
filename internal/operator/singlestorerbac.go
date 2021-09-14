// Copyright 2021 The churrodata Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package operator

import (
	"github.com/churrodata/churro/api/uiv1alpha1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r ChurrouiReconciler) processSinglestoreClusterRoles(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoles rbacv1.ClusterRoleList
	err := r.List(r.Ctx, &childRoles)
	if err != nil {
		r.Log.Error(err, "unable to list child roles for singlestore operator")
		return err
	}

	// compare referenced rbac objects with what we expect
	needSinglestoreRole := true
	for i := 0; i < len(childRoles.Items); i++ {
		r := childRoles.Items[i]
		if r.Name == "memsql-operator" {
			needSinglestoreRole = false
		}
	}
	if needSinglestoreRole {
		r.Log.Info("need memsql-operator clusterrole")
		role := rbacv1.ClusterRole{}
		role.Name = "memsql-operator"
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRole
		metadata:
		  name: memsql-operator
		rules:
		- apiGroups:
		  - ""
		  resources:
		  - pods
		  - services
		  - endpoints
		  - persistentvolumeclaims
		  - events
		  - configmaps
		  - secrets
		  verbs:
		  - '*'
		- apiGroups:
		  - policy
		  resources:
		  - poddisruptionbudgets
		  verbs:
		  - '*'
		- apiGroups:
		  - batch
		  resources:
		  - cronjobs
		  verbs:
		  - '*'
		- apiGroups:
		  - ""
		  resources:
		  - namespaces
		  verbs:
		  - get
		- apiGroups:
		  - apps
		  - extensions
		  resources:
		  - deployments
		  - daemonsets
		  - replicasets
		  - statefulsets
		  verbs:
		  - '*'
		- apiGroups:
		  - memsql.com
		  resources:
		  - '*'
		  verbs:
		  - '*'
		*/
		role.Rules = make([]rbacv1.PolicyRule, 0)
		policyRule := rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"pods", "services", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"policy"}
		policyRule.Resources = []string{"poddisruptionbudgets"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"batch"}
		policyRule.Resources = []string{"cronjobs"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"get"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"namespaces"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"apps", "extensions"}
		policyRule.Resources = []string{"deployments", "daemonsets", "replicasets", "statefulsets"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"memsql.com"}
		policyRule.Resources = []string{"*"}
		role.Rules = append(role.Rules, policyRule)

		if err := r.Create(r.Ctx, &role); err != nil {
			r.Log.Error(err, "unable to create memsql-operator clusterrole ", "rbac", role)
			return err
		}
		r.Log.Info("created memsql-operator clusterrole")
	}

	return nil
}

func (r ChurrouiReconciler) processSinglestoreServiceAccounts(uiInstance uiv1alpha1.Churroui) (err error) {
	/**
	apiVersion: v1
	kind: ServiceAccount
	metadata:
	  name: memsql-operator
	  namespace: churro
	*/

	var childSAs v1.ServiceAccountList
	err = r.List(r.Ctx, &childSAs, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child serviceaccounts for ui instance")
		return err
	}

	needSinglestoreSA := true
	for i := 0; i < len(childSAs.Items); i++ {
		r := childSAs.Items[i]
		if r.Name == "memsql-operator" {
			needSinglestoreSA = false
		}
	}
	if needSinglestoreSA {
		sa := v1.ServiceAccount{}
		sa.Name = "memsql-operator"
		sa.Namespace = uiInstance.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&uiInstance, &sa, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &sa); err != nil {
			r.Log.Error(err, "unable to create singlestore operator service account", "rbac", sa)
			return err
		}
		r.Log.Info("created memsql-operator service account")
	}
	return nil

}

func (r ChurrouiReconciler) processSinglestoreClusterRoleBindings(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoleBindings rbacv1.ClusterRoleBindingList
	if err := r.List(r.Ctx, &childRoleBindings); err != nil {
		r.Log.Error(err, "unable to list child clusterrolebindings")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a clusterrolebinding/churro-ui
	needBinding := true
	for i := 0; i < len(childRoleBindings.Items); i++ {
		r := childRoleBindings.Items[i]
		if r.Name == "memsql-operator" {
			needBinding = false
		}
	}

	// create any expected rbac objects, set owner reference to this ui
	if needBinding {
		binding := rbacv1.ClusterRoleBinding{}
		binding.Name = "memsql-operator"
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRoleBinding
		metadata:
		  name: memsql-operator
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: ClusterRole
		  name: memsql-operator
		subjects:
		  - kind: ServiceAccount
		    name: memsql-operator
		    namespace: churro
		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "memsql-operator"
		subject.Namespace = uiInstance.ObjectMeta.Namespace
		binding.Subjects = []rbacv1.Subject{subject}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "memsql-operator"}
		binding.ObjectMeta.Name = "memsql-operator"

		if err := r.Create(r.Ctx, &binding); err != nil {
			r.Log.Error(err, "unable to create memsql-operator clusterrolebinding", "rbac", binding)
			return err
		}
		r.Log.Info("created memsql-operator clusterrolebinding")
	}

	return nil
}
