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
	"context"
	"os"

	kerrors "errors"

	"github.com/churrodata/churro/api/uiv1alpha1"
	"github.com/churrodata/churro/pkg"
	"github.com/go-logr/logr"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml"
)

var (
	uiownerKey = "ui.metadata.controller"
	uiapiGVStr = uiv1alpha1.GroupVersion.String()
)

// this code is for the operator to deploy the churro web consoles

// ChurrouiReconciler ...
type ChurrouiReconciler struct {
	client.Client
	Log                         logr.Logger
	Ctx                         context.Context
	Scheme                      *runtime.Scheme
	UIDeploymentTemplate        []byte
	CockroachDBCRDTemplate      []byte
	CockroachDBOperatorTemplate []byte
	SinglestoreCRDTemplate      []byte
	SinglestoreOperatorTemplate []byte
	PVCTemplate                 []byte
}

// Reconcile ...
func (r *ChurrouiReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//r.Ctx = context.Background()
	r.Ctx = ctx
	r.Log = r.Log.WithValues("churroui", req.NamespacedName)

	// your logic here
	result, err := r.ProcessCR(req)
	if err != nil {
		return ctrl.Result{}, err
	}

	return result, nil
}

// SetupWithManager ...
func (r *ChurrouiReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log.Info("churroUI SetupWithManager...")
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Service{}, uiownerKey, func(rawObj client.Object) []string {
		dep := rawObj.(*v1.Service)
		owner := metav1.GetControllerOf(dep)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != uiapiGVStr || owner.Kind != "Churroui" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, uiownerKey, func(rawObj client.Object) []string {
		dep := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(dep)
		if owner == nil {
			return nil
		}
		// ...make sure it's a pipeline...
		if owner.APIVersion != uiapiGVStr || owner.Kind != "Churroui" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &rbacv1.ClusterRole{}, uiownerKey, func(rawObj client.Object) []string {
		// grab the role object, extract the owner...
		role := rawObj.(*rbacv1.ClusterRole)
		owner := metav1.GetControllerOf(role)
		if owner == nil {
			return nil
		}
		// ...make sure it's a pipeline...
		if owner.APIVersion != uiapiGVStr || owner.Kind != "Churroui" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &rbacv1.ClusterRoleBinding{}, uiownerKey, func(rawObj client.Object) []string {
		// grab the rolebinding object, extract the owner...
		rolebinding := rawObj.(*rbacv1.ClusterRoleBinding)
		owner := metav1.GetControllerOf(rolebinding)
		if owner == nil {
			return nil
		}
		// ...make sure it's a pipeline...
		if owner.APIVersion != uiapiGVStr || owner.Kind != "Churroui" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.ServiceAccount{}, uiownerKey, func(rawObj client.Object) []string {
		// grab the ServiceAccount object, extract the owner...
		sa := rawObj.(*v1.ServiceAccount)
		owner := metav1.GetControllerOf(sa)
		if owner == nil {
			return nil
		}
		// ...make sure it's a pipeline...
		if owner.APIVersion != uiapiGVStr || owner.Kind != "Churroui" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.PersistentVolumeClaim{}, uiownerKey, func(rawObj client.Object) []string {
		sa := rawObj.(*v1.PersistentVolumeClaim)
		owner := metav1.GetControllerOf(sa)
		if owner == nil {
			return nil
		}
		// ...make sure it's a pipeline...
		if owner.APIVersion != uiapiGVStr || owner.Kind != "Churroui" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&uiv1alpha1.Churroui{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&v1.Service{}).
		Owns(&v1.ServiceAccount{}).
		Owns(&v1.PersistentVolumeClaim{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r ChurrouiReconciler) processServiceAccounts(uiInstance uiv1alpha1.Churroui) (err error) {
	/**
	  	apiVersion: v1
	      kind: ServiceAccount
	      metadata:
	        name: churro-ui
	*/

	var childSAs v1.ServiceAccountList
	err = r.List(r.Ctx, &childSAs, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child serviceaccounts for ui instance")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a sa/churro-ui
	needChurroSA := true
	for i := 0; i < len(childSAs.Items); i++ {
		r := childSAs.Items[i]
		if r.Name == "churro-ui" {
			needChurroSA = false
		}
	}

	if needChurroSA {
		churroSA := v1.ServiceAccount{}
		churroSA.Name = "churro-ui"
		churroSA.Namespace = uiInstance.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&uiInstance, &churroSA, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &churroSA); err != nil {
			r.Log.Error(err, "unable to create service account for ui", "rbac", churroSA)
			return err
		}
		log.Info().Msg("created service account for churro-ui ")
	}
	return nil

}

// ProcessCR handles a reconcile event
func (r ChurrouiReconciler) ProcessCR(req ctrl.Request) (result ctrl.Result, err error) {

	// get the CR
	var uiInstance uiv1alpha1.Churroui
	if err := r.Get(r.Ctx, req.NamespacedName, &uiInstance); err != nil {
		r.Log.Error(err, "unable to fetch churroUI")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.

		// clean up the namespace we created for the pipeline
		/**
		ns := v1.Namespace{}
		ns.Name = req.Namespace

		if err := r.Delete(r.Ctx, &ns); err != nil {
			log.Error(err, "unable to delete Namespace for pipeline", "namespace", ns)
			return result, err
		}
		*/

		return result, client.IgnoreNotFound(err)
	}
	r.Log.Info("got a churroUI " + uiInstance.Name)
	err = r.processChurroui(req, uiInstance)
	if err != nil {
		r.Log.Error(err, "error in processing churroUI")
	}
	return result, err
}

func (r ChurrouiReconciler) processChurroui(req ctrl.Request, uiInstance uiv1alpha1.Churroui) error {

	err := r.processServiceAccounts(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Error(err, "error in processing ServiceAccounts")
		}
	}
	err = r.processClusterRoles(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Error(err, "error in processing ClusterRole")
		}
	}
	err = r.processClusterRoleBindings(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Error(err, "error in processing ClusterRoleBinding")
			//return err
		}
	}
	err = r.processService(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Error(err, "error in processing Service")
			//return err
		}
	}
	err = r.processPVCs(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Error(err, "error in processing PVC")
			//return err
		}
	}
	err = r.processDeployment(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			r.Log.Error(err, "error in processing Deployment")
			//return err
		}
	}
	/**
	err = r.processCRDs(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing CRDs")
		}
	}
	err = r.processOperatorDeployment(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing operator Deployment")
			//return err
		}
	}
	err = r.processCockroachClusterRoles(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing cockroach clusterroles")
			//return err
		}
	}
	err = r.processCockroachServiceAccounts(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing cockroach serviceaccounts")
			//return err
		}
	}
	err = r.processCockroachClusterRoleBindings(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing cockroach clusterrolebindings")
			//return err
		}
	}
	err = r.processCockroachRoles(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing cockroach roles")
			//return err
		}
	}
	err = r.processCockroachRoleBindings(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing cockroach rolebindings")
			//return err
		}
	}
	err = r.processSinglestoreClusterRoles(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing SinglestoreClusterRoles ")
			//return err
		}
	}
	err = r.processSinglestoreServiceAccounts(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing SinglestoreServiceAccounts ")
			//return err
		}
	}
	err = r.processSinglestoreClusterRoleBindings(uiInstance)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "error in processing singlestore ClusterRoleBindings ")
			//return err
		}
	}
	*/
	return nil
}

// Package operator holds the churro operator logic
func (r ChurrouiReconciler) processClusterRoles(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoles rbacv1.ClusterRoleList
	//err := r.List(r.Ctx, &childRoles, client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	err := r.List(r.Ctx, &childRoles)
	if err != nil {
		r.Log.Error(err, "unable to list child roles for ui roles")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a clusterrole/churro-ui
	needChurroRole := true
	for i := 0; i < len(childRoles.Items); i++ {
		r := childRoles.Items[i]
		if r.Name == "churro-ui" {
			needChurroRole = false
		}
	}

	// create any expected rbac objects, set owner reference to this churroui
	//rolesToCreate := make([]rbacv1.ClusterRole, 0)
	if needChurroRole {
		log.Info().Msg(" need ui clusterrole")
		churroRole := rbacv1.ClusterRole{}
		churroRole.Name = "churro-ui"
		//churroRole.Namespace = uiInstance.ObjectMeta.Namespace
		/**
				apiVersion: rbac.authorization.k8s.io/v1
		kind: ClusterRole
		metadata:
		  name: churro-ui
		rules:
		- apiGroups:
		  - churro.project.io
		  resources:
		  - pipelines
		  verbs:
		  - create
		  - delete
		  - get
		  - list
		  - patch
		  - update
		  - watch
		- apiGroups:
		  - ''
		  resources:
		  - pods
		  - pods/log
		  - namespaces
		  verbs:
		  - list
		  - get
		  - create
		*/
		churroRole.Rules = make([]rbacv1.PolicyRule, 0)
		policyRule := rbacv1.PolicyRule{
			Verbs:     []string{"create", "get", "list", "delete", "patch", "update", "watch"},
			APIGroups: []string{"churro.project.io"},
			Resources: []string{"pipelines"},
		}
		churroRole.Rules = append(churroRole.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"get", "list", "create"},
			APIGroups: []string{""},
			Resources: []string{"pods", "pods/log", "namespaces"},
		}
		churroRole.Rules = append(churroRole.Rules, policyRule)

		policyRule = rbacv1.PolicyRule{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
		}
		churroRole.Rules = append(churroRole.Rules, policyRule)

		log.Info().Msg(", createing clusterrole for ui...")
		//if err := ctrl.SetControllerReference(&uiInstance, &churroRole, r.Scheme); err != nil {
		//return err
		//}
		if err := r.Create(r.Ctx, &churroRole); err != nil {
			r.Log.Error(err, "unable to create rbacObject for ui", "rbac", churroRole)
			return err
		}
		log.Info().Msg("created clusterrole for ui ")
	}

	return nil
}

func (r ChurrouiReconciler) processClusterRoleBindings(uiInstance uiv1alpha1.Churroui) error {
	// get referenced rbac objects
	var childRoleBindings rbacv1.ClusterRoleBindingList
	//if err := r.List(r.Ctx, &childRoleBindings, client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name}); err != nil {
	if err := r.List(r.Ctx, &childRoleBindings); err != nil {
		r.Log.Error(err, "unable to list child clusterrolebindings")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a clusterrolebinding/churro-ui
	needChurroRoleBinding := true
	for i := 0; i < len(childRoleBindings.Items); i++ {
		r := childRoleBindings.Items[i]
		if r.Name == "churro-ui" {
			needChurroRoleBinding = false
		}
	}

	// create any expected rbac objects, set owner reference to this ui
	roleBindingsToCreate := make([]rbacv1.ClusterRoleBinding, 0)
	if needChurroRoleBinding {
		churroRoleBinding := rbacv1.ClusterRoleBinding{}
		churroRoleBinding.Name = "churro-ui"
		//churroRoleBinding.Namespace = uiInstance.ObjectMeta.Namespace
		/**
				kind: ClusterRoleBinding
		apiVersion: rbac.authorization.k8s.io/v1
		metadata:
		  name: churro-ui
		roleRef:
		  apiGroup: rbac.authorization.k8s.io
		  kind: ClusterRole
		  name: churro-ui
		subjects:
		- apiGroup: rbac.authorization.k8s.io
		  kind: User
		  name: system:serviceaccount:churro:churro-ui

		*/
		subject := rbacv1.Subject{}
		subject.Kind = "ServiceAccount"
		subject.Name = "churro-ui"
		subject.Namespace = uiInstance.ObjectMeta.Namespace
		churroRoleBinding.Subjects = []rbacv1.Subject{subject}
		churroRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "churro-ui"}
		churroRoleBinding.ObjectMeta.Name = subject.Name
		churroRoleBinding.ObjectMeta.Labels = map[string]string{"app": subject.Name}
		roleBindingsToCreate = append(roleBindingsToCreate, churroRoleBinding)
	}

	for _, roleBinding := range roleBindingsToCreate {
		//if err := ctrl.SetControllerReference(&uiInstance, &roleBinding, r.Scheme); err != nil {
		//return err
		//}
		if err := r.Create(r.Ctx, &roleBinding); err != nil {
			r.Log.Error(err, "unable to create roleBinding for ui", "rbac", roleBinding)
			return err
		}
		log.Info().Msg("created rolebinding for ui ")
	}

	return nil
}

func (r ChurrouiReconciler) processService(uiInstance uiv1alpha1.Churroui) (err error) {
	/**
	  	apiVersion: v1
	      kind: Service
	      metadata:
	        name: churro-ui
	      spec:
	        selector:
	          app: churro-ui
	        ports:
	          - protocol: TCP
	            port: 8080
	            targetPort: 8080
	*/

	var childServices v1.ServiceList
	err = r.List(r.Ctx, &childServices, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child services for ui instance")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a service/churro-ui
	needChurroService := true
	for i := 0; i < len(childServices.Items); i++ {
		r := childServices.Items[i]
		if r.Name == "churro-ui" {
			needChurroService = false
		}
	}

	if needChurroService {
		serviceType := v1.ServiceTypeClusterIP
		log.Info().Msg("servicetype  " + uiInstance.Spec.ServiceType)
		if uiInstance.Spec.ServiceType == "LoadBalancer" {
			log.Info().Msg("setting service type to LoadBalancer")
			serviceType = v1.ServiceTypeLoadBalancer
		}
		churroService := v1.Service{}
		churroService.Name = "churro-ui"
		churroService.Namespace = uiInstance.ObjectMeta.Namespace
		churroService.Spec = v1.ServiceSpec{
			Selector: map[string]string{
				"app": "churro-ui",
			},
			Type: serviceType,
			Ports: []v1.ServicePort{
				{
					Port: 8080,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
		}

		if err := ctrl.SetControllerReference(&uiInstance, &churroService, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &churroService); err != nil {
			r.Log.Error(err, "unable to create service for ui", "rbac", churroService)
			return err
		}
		log.Info().Msg("created service for churro-ui ")
	}
	return nil

}

func (r ChurrouiReconciler) processPVCs(uiInstance uiv1alpha1.Churroui) (err error) {
	/**
	apiVersion: v1
	kind: PersistentVolumeClaim
	metadata:
	  name: churro-admindb
	spec:
	  storageClassName: hostpath
	  accessModes:
	    - ReadWriteOnce
	  resources:
	    requests:
	      storage: 1Gi
	*/

	var childPVCs v1.PersistentVolumeClaimList
	err = r.List(r.Ctx, &childPVCs, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child pvcs for ui instance")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a sa/churro-ui
	needChurroPVC := true
	for i := 0; i < len(childPVCs.Items); i++ {
		r := childPVCs.Items[i]
		if r.Name == "churro-admindb" {
			needChurroPVC = false
		}
	}

	if needChurroPVC {
		var churroPVC v1.PersistentVolumeClaim
		err := yaml.Unmarshal(r.PVCTemplate, &churroPVC)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal PVC template")
			return err
		}

		churroPVC.Name = "churro-admindb"
		churroPVC.Namespace = uiInstance.ObjectMeta.Namespace

		v := getDefaultSC(r.Log)
		churroPVC.Spec.StorageClassName = &v

		if uiInstance.Spec.StorageClassName != "" {
			churroPVC.Spec.StorageClassName = &uiInstance.Spec.StorageClassName
		}
		if uiInstance.Spec.StorageSize != "" {
			qty, err := resource.ParseQuantity(uiInstance.Spec.StorageSize)
			if err != nil {
				r.Log.Error(err, "unable to parse storage size")
				return err
			}
			churroPVC.Spec.Resources.Requests[v1.ResourceStorage] = qty
		}

		switch uiInstance.Spec.AccessMode {
		case "", "ReadWriteMany":
			churroPVC.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteMany}
		case "ReadWriteOnce":
			churroPVC.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
		default:
			r.Log.Error(kerrors.New("accessmode is not right"+uiInstance.Spec.AccessMode), "error in AccessMode")
			return err
		}

		if err := ctrl.SetControllerReference(&uiInstance, &churroPVC, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &churroPVC); err != nil {
			r.Log.Error(err, "unable to create persistent volume claim for ui", "rbac", churroPVC)
			return err
		}
		log.Info().Msg("created persistent volume claim for churro-ui ")
	}
	return nil

}

func (r ChurrouiReconciler) processDeployment(uiInstance uiv1alpha1.Churroui) error {
	// get referenced Deployment objects
	var children appsv1.DeploymentList
	err := r.List(r.Ctx, &children, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child ui Deployments")
		return err
	}

	// compare referenced Deployment objects with what we expect
	// make sure we have a deployment/churro-ui
	needDeployment := true

	for i := 0; i < len(children.Items); i++ {
		r := children.Items[i]
		switch r.Name {
		case "churro-ui":
			needDeployment = false
		}
	}

	// create any expected Deployment objects, set owner reference to this ui instance
	if needDeployment {
		var deployment appsv1.Deployment
		err := yaml.Unmarshal(r.UIDeploymentTemplate, &deployment)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal ui Deployment template")
			return err
		}

		pullSecretName := os.Getenv("CHURRO_PULL_SECRET_NAME")
		if pullSecretName != "" {
			ref := v1.LocalObjectReference{}
			ref.Name = pullSecretName
			deployment.Spec.Template.Spec.ImagePullSecrets = []v1.LocalObjectReference{ref}
		}

		deployment.ObjectMeta.Labels = map[string]string{"app": "churro-ui"}
		deployment.Name = "churro-ui"
		deployment.Namespace = uiInstance.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&uiInstance, &deployment, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &deployment); err != nil {
			r.Log.Error(err, "unable to create ui deployment ", "deployment", deployment)
			return err
		}
		r.Log.V(1).Info("created deployment for ui ")
	}

	return nil
}

func (r ChurrouiReconciler) processCRDs(uiInstance uiv1alpha1.Churroui) error {

	_, config, err := pkg.GetKubeClient()
	if err != nil {
		log.Error().Stack().Err(err).Msg("some error")
	}

	xClient, err := apiextension.NewForConfig(config)
	if err != nil {
		r.Log.Error(err, "unable to connect to  CRDs")
		return err
	}
	crds, err := xClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		r.Log.Error(err, "unable to list CRDs")
		return err
	}

	// compare referenced rbac objects with what we expect
	// make sure we have a customresourcedefinition/crdbclusters.crdb.cockroachlabs.com
	// make sure we have a customresourcedefinition/memsqlclusters.memsql.com
	needCockroachCRD := true
	needSinglestoreCRD := true
	for i := 0; i < len(crds.Items); i++ {
		r := crds.Items[i]
		if r.Name == "memsqlclusters.memsql.com" {
			needSinglestoreCRD = false
		}
		if r.Name == "crdbclusters.crdb.cockroachlabs.com" {
			needCockroachCRD = false
		}
	}

	// create any expected CRD objects
	if needCockroachCRD {
		log.Info().Msg(", need cockroachdb crd\n")
		var crd apiextv1.CustomResourceDefinition

		err = yaml.Unmarshal(r.CockroachDBCRDTemplate, &crd)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal cockroachdb CRD template")
			return err
		}

		createOptions := metav1.CreateOptions{}
		_, err = xClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), &crd, createOptions)
		if err != nil {
			r.Log.Error(err, "unable to create CRD")
			return err
		}

		log.Info().Msg("created crd for cockroach ")
	}

	if needSinglestoreCRD {
		log.Info().Msg(", need singlestore crd\n")
		var crd apiextv1.CustomResourceDefinition
		err = yaml.Unmarshal(r.SinglestoreCRDTemplate, &crd)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal singlestore CRD template")
			return err
		}

		log.Info().Msg(", creating crd for singlestore...")
		createOptions := metav1.CreateOptions{}
		_, err = xClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), &crd, createOptions)
		if err != nil {
			r.Log.Error(err, "unable to create CRD")
			return err
		}
		log.Info().Msg("created crd for singlestore ")
	}

	return nil
}

func (r ChurrouiReconciler) processOperatorDeployment(uiInstance uiv1alpha1.Churroui) error {
	// get referenced Deployment objects
	var children appsv1.DeploymentList
	err := r.List(r.Ctx, &children, client.InNamespace(uiInstance.ObjectMeta.Namespace), client.MatchingFields{uiownerKey: uiInstance.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child ui Deployments")
		return err
	}

	// compare referenced Deployment objects with what we expect
	// make sure we have a deployment/cockroach-operator
	needCockroachDeployment := true
	needSinglestoreDeployment := true

	for i := 0; i < len(children.Items); i++ {
		r := children.Items[i]
		if r.Name == "cockroach-operator" {
			needCockroachDeployment = false
		} else if r.Name == "memsql-operator" {
			needSinglestoreDeployment = false
		}
	}

	// create any expected Deployment objects, set owner reference to this ui instance
	if needCockroachDeployment {
		var deployment appsv1.Deployment
		err := yaml.Unmarshal(r.CockroachDBOperatorTemplate, &deployment)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal cockroach-operator Deployment template")
			return err
		}

		deployment.Namespace = uiInstance.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&uiInstance, &deployment, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &deployment); err != nil {
			r.Log.Error(err, "unable to create cockroach-operator deployment ", "deployment", deployment)
			return err
		}
		r.Log.V(1).Info("created deployment for cockroach-operator ")
	}
	if needSinglestoreDeployment {
		var deployment appsv1.Deployment
		err := yaml.Unmarshal(r.SinglestoreOperatorTemplate, &deployment)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal singlestore-operator Deployment template")
			return err
		}

		deployment.Namespace = uiInstance.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&uiInstance, &deployment, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &deployment); err != nil {
			r.Log.Error(err, "unable to create memsql-operator deployment ", "deployment", deployment)
			return err
		}
		r.Log.V(1).Info("created deployment for memsql-operator ")
	}

	return nil
}
