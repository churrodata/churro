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
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r ChurrouiReconciler) processCockroachClusterRoles(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoles rbacv1.ClusterRoleList
	err := r.List(r.Ctx, &childRoles)
	if err != nil {
		r.Log.Error(err, "unable to list child roles for ui roles")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a clusterrole/cockroach-database-role
	needCockroachDatabaseRole := true
	needCockroachOperatorRole := true
	for i := 0; i < len(childRoles.Items); i++ {
		r := childRoles.Items[i]
		if r.Name == "cockroach-database-role" {
			needCockroachDatabaseRole = false
		} else if r.Name == "cockroach-operator-role" {
			needCockroachOperatorRole = false
		}
	}
	if needCockroachDatabaseRole {
		log.Info().Msg(", need cockroach-database-role clusterrole")
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRole
		metadata:
		  name: cockroach-database-role
		rules:
		  - verbs:
		      - use
		    apiGroups:
		      - security.openshift.io
		    resources:
		      - securitycontextconstraints
		    resourceNames:
		      - anyuid
		*/
		role := rbacv1.ClusterRole{}
		role.Name = "cockroach-database-role"
		role.Rules = make([]rbacv1.PolicyRule, 0)
		policyRule := rbacv1.PolicyRule{
			Verbs:         []string{"use"},
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			ResourceNames: []string{"anyuid"},
		}
		role.Rules = append(role.Rules, policyRule)

		if err := r.Create(r.Ctx, &role); err != nil {
			r.Log.Error(err, "unable to create cockroach-database-role clusterrole ", "rbac", role)
			return err
		}
		log.Info().Msg("created cockroach-database-role clusterrole")
	}
	if needCockroachOperatorRole {
		log.Info().Msg(", need cockroach-operator-role clusterrole")
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRole
		metadata:
		  creationTimestamp: null
		  name: cockroach-operator-role
		rules:
		  - apiGroups:
		      - "*"
		    resources:
		      - "*"
		    verbs:
		      - "*"
		  - apiGroups:
		      - rbac.authorization.k8s.io
		    resources:
		      - clusterroles
		    verbs:
		      - get
		      - list
		      - delete
		  - apiGroups:
		      - apps
		    resources:
		      - statefulsets
		    verbs:
		      - "*"
		  - apiGroups:
		      - apps
		    resources:
		      - statefulsets/finalizers
		    verbs:
		      - "*"
		 - apiGroups:
		      - apps
		    resources:
		      - statefulsets/status
		    verbs:
		      - "*"
		  - apiGroups:
		      - certificates.k8s.io
		    resources:
		      - certificatesigningrequests
		    verbs:
		      - "*"
		  - apiGroups:
		      - certificates.k8s.io
		    resources:
		      - certificatesigningrequests/approval
		    verbs:
		      - "*"
		  - apiGroups:
		      - certificates.k8s.io
		    resources:
		      - certificatesigningrequests/status
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - configmaps
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - configmaps/status
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - pods/exec
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - secrets
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - services
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - services/finalizers
		    verbs:
		      - "*"
		  - apiGroups:
		      - ""
		    resources:
		      - services/status
		    verbs:
		      - "*"
		  - apiGroups:
		      - crdb.cockroachlabs.com
		    resources:
		      - crdbclusters
		    verbs:
		      - "*"
		  - apiGroups:
		      - crdb.cockroachlabs.com
		    resources:
		      - crdbclusters/status
		    verbs:
		      - "*"
		  - apiGroups:
		      - policy
		    resources:
		      - poddisruptionbudgets
		    verbs:
		      - "*"
		  - apiGroups:
		      - policy
		    resources:
		      - poddisruptionbudgets/finalizers
		    verbs:
		      - "*"
		  - apiGroups:
		      - policy
		    resources:
		      - poddisruptionbudgets/status
		    verbs:
		      - "*"
		  - verbs:
		      - use
		    apiGroups:
		      - security.openshift.io
		    resources:
		      - securitycontextconstraints
		    resourceNames:
		      - anyuid
		*/
		role := rbacv1.ClusterRole{}
		role.Name = "cockroach-operator-role"
		role.Rules = make([]rbacv1.PolicyRule, 0)

		policyRule := rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"*"},
			Resources: []string{"*"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"get", "list", "delete"},
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"clusterroles"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets/finalizers"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets/status"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"certificates.k8s.io"},
			Resources: []string{"certificatesigningrequests"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"certificates.k8s.io"},
			Resources: []string{"certificatesigningrequests/approval"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"*"},
			APIGroups: []string{"certificates.k8s.io"},
			Resources: []string{"certificatesigningrequests/status"},
		}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"configmaps"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"configmaps/status"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"pods/exec"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"secrets"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"services"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{""}
		policyRule.Resources = []string{"services/status"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"crdb.cockroachlabs.com"}
		policyRule.Resources = []string{"crdbclusters"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"crdb.cockroachlabs.com"}
		policyRule.Resources = []string{"crdbclusters/status"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"policy"}
		policyRule.Resources = []string{"poddisruptionbudgets"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"policy"}
		policyRule.Resources = []string{"poddisruptionbudgets/finalizers"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"policy"}
		policyRule.Resources = []string{"poddisruptionbudgets/status"}
		role.Rules = append(role.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"use"}
		policyRule.APIGroups = []string{"security.openshift.io"}
		policyRule.Resources = []string{"securitycontextconstraints"}
		policyRule.ResourceNames = []string{"anyuid"}
		role.Rules = append(role.Rules, policyRule)

		if err := r.Create(r.Ctx, &role); err != nil {
			r.Log.Error(err, "unable to create cockroach-database-role clusterrole ", "rbac", role)
			return err
		}
		log.Info().Msg("created cockroach-database-role clusterrole")
	}

	return nil
}

func (r ChurrouiReconciler) processCockroachServiceAccounts(uiInstance uiv1alpha1.Churroui) (err error) {
	/**
	apiVersion: v1
	kind: ServiceAccount
	metadata:
	  name: cockroach-database-sa
	  namespace: churro
	  annotations:
	  labels:
	    app: cockroach-operator
	*/

	var childSAs v1.ServiceAccountList
	err = r.List(r.Ctx, &childSAs, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child serviceaccounts for ui instance")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a sa/churro-ui
	needCockroachDatabaseSA := true
	needCockroachOperatorSA := true
	for i := 0; i < len(childSAs.Items); i++ {
		r := childSAs.Items[i]
		if r.Name == "cockroach-database-sa" {
			needCockroachDatabaseSA = false
		} else if r.Name == "cockroach-operator-sa" {
			needCockroachOperatorSA = false
		}
	}
	if needCockroachDatabaseSA {
		sa := v1.ServiceAccount{}
		sa.Name = "cockroach-database-sa"
		sa.Namespace = uiInstance.ObjectMeta.Namespace
		sa.ObjectMeta.Labels = map[string]string{"app": "cockroach-operator"}

		if err := ctrl.SetControllerReference(&uiInstance, &sa, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &sa); err != nil {
			r.Log.Error(err, "unable to create cockroach-database-sa service account", "rbac", sa)
			return err
		}
		log.Info().Msg("created cockroach-database-sa service account")
	}
	if needCockroachOperatorSA {
		sa := v1.ServiceAccount{}
		sa.Name = "cockroach-operator-sa"
		sa.Namespace = uiInstance.ObjectMeta.Namespace
		sa.ObjectMeta.Labels = map[string]string{"app": "cockroach-operator"}

		if err := ctrl.SetControllerReference(&uiInstance, &sa, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &sa); err != nil {
			r.Log.Error(err, "unable to create cockroach-operator-sa service account", "rbac", sa)
			return err
		}
		log.Info().Msg("created cockroach-operator-sa service account")
	}
	return nil

}

func (r ChurrouiReconciler) processCockroachClusterRoleBindings(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoleBindings rbacv1.ClusterRoleBindingList
	if err := r.List(r.Ctx, &childRoleBindings); err != nil {
		r.Log.Error(err, "unable to list child clusterrolebindings")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a clusterrolebinding/churro-ui
	needCockroachDatabaseRoleBinding := true
	needCockroachOperatorRoleBinding := true
	for i := 0; i < len(childRoleBindings.Items); i++ {
		r := childRoleBindings.Items[i]
		if r.Name == "cockroach-database-rolebinding" {
			needCockroachDatabaseRoleBinding = false
		} else if r.Name == "cockroach-operator-role" {
			needCockroachOperatorRoleBinding = false
		}
	}

	// create any expected rbac objects, set owner reference to this ui
	if needCockroachDatabaseRoleBinding {
		binding := rbacv1.ClusterRoleBinding{}
		binding.Name = "cockroach-database-rolebinding"
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRoleBinding
		metadata:
		  name: cockroach-database-rolebinding
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: ClusterRole
		  name: cockroach-database-role
		subjects:
		  - kind: ServiceAccount
		    name: cockroach-database-sa
		    namespace: churro
		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "cockroach-database-sa"
		subject.Namespace = uiInstance.ObjectMeta.Namespace
		binding.Subjects = []rbacv1.Subject{subject}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cockroach-database-role"}
		binding.ObjectMeta.Name = "cockroach-database-rolebinding"

		if err := r.Create(r.Ctx, &binding); err != nil {
			r.Log.Error(err, "unable to create cockroach-database-rolebinding clusterrolebinding", "rbac", binding)
			return err
		}
		log.Info().Msg("created cockroach-database-rolebinding clusterrolebinding")
	}

	if needCockroachOperatorRoleBinding {
		binding := rbacv1.ClusterRoleBinding{}
		binding.Name = "cockroach-operator-rolebinding"
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRoleBinding
		metadata:
		  name: cockroach-operator-rolebinding
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: ClusterRole
		  name: cockroach-operator-role
		subjects:
		  - kind: ServiceAccount
		    name: cockroach-operator-sa
		    namespace: churro

		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "cockroach-operator-sa"
		subject.Namespace = uiInstance.ObjectMeta.Namespace
		binding.Subjects = []rbacv1.Subject{subject}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cockroach-operator-role"}
		binding.ObjectMeta.Name = "cockroach-operator-rolebinding"
		if err := r.Create(r.Ctx, &binding); err != nil {
			r.Log.Error(err, "unable to create cockroach-operator-rolebinding clusterrolebinding", "rbac", binding)
			return err
		}
		log.Info().Msg("created cockroach-operator-rolebinding clusterrolebinding")
	}

	return nil
}

func (r ChurrouiReconciler) processCockroachRoles(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoles rbacv1.RoleList
	err := r.List(r.Ctx, &childRoles, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child roles")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a role/churro
	needRole := true
	for i := 0; i < len(childRoles.Items); i++ {
		r := childRoles.Items[i]
		if r.Name == "cockroach-operator-role" {
			needRole = false
		}
	}

	// create any expected rbac objects, set owner reference to this uiInstance
	if needRole {
		role := rbacv1.Role{}
		role.Name = "cockroach-operator-role"
		role.Namespace = uiInstance.ObjectMeta.Namespace
		/**
		apiVersion: rbac.authorization.k8s.io/v1
		kind: Role
		metadata:
		  name: cockroach-operator-role
		rules:
		  - apiGroups:
		      - "*"
		    resources:
		      - "*"
		    verbs:
		      - "*"
		*/
		role.Rules = make([]rbacv1.PolicyRule, 0)
		policyRule := rbacv1.PolicyRule{}
		policyRule.Verbs = []string{"*"}
		policyRule.APIGroups = []string{"*"}
		policyRule.Resources = []string{"*"}
		role.Rules = append(role.Rules, policyRule)

		if err := ctrl.SetControllerReference(&uiInstance, &role, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &role); err != nil {
			r.Log.Error(err, "unable to create cockroach-operator-role", "rbac", role)
			return err
		}
		log.Info().Msg("created cockroach-operator-role")
	}

	return nil
}

func (r ChurrouiReconciler) processCockroachRoleBindings(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoleBindings rbacv1.RoleBindingList
	if err := r.List(r.Ctx, &childRoleBindings, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: uiInstance.ObjectMeta.Name}); err != nil {
		r.Log.Error(err, "unable to list child rolebindings")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a rolebinding/churro
	needCockroachOperatorRoleBinding := true
	// make sure we have a rolebinding/cockroach-operator-churro
	for i := 0; i < len(childRoleBindings.Items); i++ {
		r := childRoleBindings.Items[i]
		if r.Name == "cockroach-operator-churro" {
			needCockroachOperatorRoleBinding = false
		}
	}

	// create any expected rbac objects, set owner reference to this uiInstance
	roleBindingsToCreate := make([]rbacv1.RoleBinding, 0)
	if needCockroachOperatorRoleBinding {
		binding := rbacv1.RoleBinding{}
		binding.Name = "cockroach-operator-churro"
		binding.Namespace = uiInstance.ObjectMeta.Namespace
		/**
		apiVersion: rbac.authorization.k8s.io/v1beta1
		kind: RoleBinding
		metadata:
		  name: cockroach-operator-churro
		  labels:
		    app: cockroach-operator
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: Role
		  name: cockroach-operator-role
		subjects:
		  - name: cockroach-operator-sa
		    namespace: churro
		    kind: ServiceAccount
		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "cockroach-operator-role"
		subject.Namespace = uiInstance.ObjectMeta.Namespace
		binding.Subjects = []rbacv1.Subject{subject}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "cockroach-operator-role"}
		binding.ObjectMeta.Name = "cockroach-operator-churro"

		binding.ObjectMeta.Labels = map[string]string{"app": "cockroach-operator"}
		roleBindingsToCreate = append(roleBindingsToCreate, binding)
	}

	for _, roleBinding := range roleBindingsToCreate {
		if err := ctrl.SetControllerReference(&uiInstance, &roleBinding, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &roleBinding); err != nil {
			r.Log.Error(err, "unable to create roleBinding", "rbac", roleBinding)
			return err
		}
		log.Info().Msg("created rolebinding for cockroach ")
	}

	return nil
}
