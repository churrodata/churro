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
	"fmt"
	"time"

	"github.com/churrodata/churro/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func (r PipelineReconciler) processSinglestore(pipeline v1alpha1.Pipeline) error {

	namespace := pipeline.ObjectMeta.Namespace
	//pipelineName := pipeline.ObjectMeta.Name

	clientset, restConfig, err := GetKubeClient()

	crBytes := r.SinglestoreCRTemplate

	// decode YAML into unstructured.Unstructured
	obj := &unstructured.Unstructured{}
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := dec.Decode([]byte(crBytes), nil, obj)
	if err != nil {
		r.Log.Error(err, "error decoding singlestore CR yaml ")
		return err
	}

	// Get the common metadata, and show GVK
	r.Log.Info(fmt.Sprintf("gvk is %s and %s", obj.GetName(), gvk.String()))

	singlestoreRes := schema.GroupVersionResource{Group: "memsql.com", Version: "v1alpha1", Resource: "memsqlclusters"}
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		r.Log.Error(err, "error getting dynamic client ")
		return err
	}

	// see if the CR exists
	list, err := dynClient.Resource(singlestoreRes).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		r.Log.Error(err, "singlestore CR list error")
		return err
	}

	crExists := false
	if len(list.Items) > 0 {
		crExists = true
	}

	if crExists == false {
		// create the CR
		r.Log.Info("Creating memsql CR...")
		result, err := dynClient.Resource(singlestoreRes).Namespace(namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
		if err != nil {
			r.Log.Error(err, "error creating singlestore CR in namespace "+namespace)
			return err
		}
		r.Log.Info(fmt.Sprintf("Created cr %q.\n", result.GetName()))

	}

	// make sure singlestore is Ready
	err = r.isSinglestoreReady(clientset, namespace)
	if err != nil {
		r.Log.Error(err, "error getting singlestore running ")
		return err
	}

	// r.Ctx

	// see if the CR exists, if not add a CR to this namespace
	//mysqlcr.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
	//mysqlcr.Namespace = pipeline.ObjectMeta.Namespace
	//	if err := ctrl.SetControllerReference(&pipeline, &mysqlcr, r.Scheme); err != nil {
	//		return err
	//	}

	// if the CR exists, see if the memsql pods are running and ready

	/**
		if err := r.Create(r.Ctx, &mysqlcr); err != nil {
			log.Error(err, "unable to create mysql cr for pipeline", "mysqlcr", mysqlcr)
			return err
		}
		r.Log.V(1).Info("created mysql cr for pipeline ")
	}
	*/

	return nil
}

func (r PipelineReconciler) isSinglestoreReady(clientset *kubernetes.Clientset, namespace string) error {
	podName := "node-memsql-cluster-master-0"

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		r.Log.Error(err, "error getting pods")
		return err
	}

	// Print its creation time
	r.Log.Info(fmt.Sprintf("pod %s was created at %s\n", podName, pod.GetCreationTimestamp()))

	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		var tmp *corev1.Pod
		tmp, err = clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Errorf("error getting pod %s %w", podName, err)
		}
		if tmp.Status.Phase == corev1.PodRunning {
			r.Log.Info(fmt.Sprintf("pod %s pod is found to be running!", podName))
			return true, nil
		}
		return false, nil
	})
	var podRunning bool
	err = wait.Poll(time.Duration(1*time.Second), time.Duration(20*time.Second), podCheck)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("timeout waiting for %s to be running...", podName))
		podRunning = false
	} else {
		podRunning = true
	}

	r.Log.Info(fmt.Sprintf(":  pod %s pod is deemed running %t\n", podName, podRunning))

	// check ready status of that pod, needs to have all containers Ready
	podReadyCheck := wait.ConditionFunc(func() (done bool, err error) {
		var tmp *corev1.Pod
		tmp, err = clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Errorf("error getting pod %s %w", podName, err)
		}
		overallReadyStatus := false
		for _, v := range tmp.Status.ContainerStatuses {
			if v.Ready {
				overallReadyStatus = true
				r.Log.Info("container found to be Ready")
			} else {
				overallReadyStatus = false
			}
		}
		return overallReadyStatus, nil
	})
	err = wait.Poll(time.Duration(1*time.Second), time.Duration(20*time.Second), podReadyCheck)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("timeout waiting for %s to be ready...", podName))
		return err
	}
	r.Log.Info("pod %s is ready..." + podName)
	return nil

}
