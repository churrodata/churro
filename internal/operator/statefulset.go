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
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/churrodata/churro/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	yaml "sigs.k8s.io/yaml"
)

func (r PipelineReconciler) processStatefulSet(pipeline v1alpha1.Pipeline) error {
	// get referenced StatefulSet objects
	var children v1.StatefulSetList
	err := r.List(r.Ctx, &children, client.InNamespace(pipeline.ObjectMeta.Namespace), client.MatchingFields{jobOwnerKey: pipeline.ObjectMeta.Name})
	if err != nil {
		r.Log.Error(err, "unable to list child StatefulSets")
		return err
	}

	// compare referenced StatefulSet objects with what we expect
	// make sure we have a statefulset/cockroachdb
	needStatefulSet := true

	for i := 0; i < len(children.Items); i++ {
		r := children.Items[i]
		switch r.Name {
		case "cockroachdb":
			needStatefulSet = false
		}
	}

	// create any expected StatefulSet objects, set owner reference to this pipeline
	if needStatefulSet {
		var statefulset v1.StatefulSet
		err := yaml.Unmarshal(r.StatefulSetTemplate, &statefulset)
		if err != nil {
			r.Log.Error(err, "unable to unmarshal StatefulSet template")
			return err
		}

		statefulset.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
		statefulset.Namespace = pipeline.ObjectMeta.Namespace

		if err := ctrl.SetControllerReference(&pipeline, &statefulset, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(r.Ctx, &statefulset); err != nil {
			r.Log.Error(err, "unable to create statefulset for pipeline", "statefulset", statefulset)
			return err
		}
		r.Log.V(1).Info("created statefulset for pipeline ")
	}

	return nil
}

func (r PipelineReconciler) initStatefulSet(req ctrl.Request, pipeline v1alpha1.Pipeline) error {
	//r.Log.V(1).Info("initStatefulSet called")
	r.Log.Info("initStatefulSet called")
	// get StatefulSet pods objects
	labelSelector, err := labels.Parse("statefulset.kubernetes.io/pod-name=cockroachdb-0")
	if err != nil {
		r.Log.Error(err, "unable to parse label selector")
		return err
	}
	listOpts := &client.ListOptions{LabelSelector: labelSelector}
	var children corev1.PodList
	err = r.List(r.Ctx, &children, client.InNamespace(pipeline.ObjectMeta.Namespace), listOpts)
	if err != nil {
		r.Log.Error(err, "unable to list child pods for StatefulSets")
		return err
	}

	// compare referenced StatefulSet objects with what we expect
	// make sure we have a statefulset/cockroachdb
	podRunning := false

	/**
	for i := 0; i < len(children.Items); i++ {
		r := children.Items[i]
		switch r.Name {
		case "cockroachdb-0":
			if r.Status.Phase == corev1.PodRunning {
				podRunning = true
				log.Info("found cockroachdb-0 pod in running status phase")
			}
		}
	}
	*/

	podName := "cockroachdb-0"

	clientset, config, err := GetKubeClient()
	if err != nil {
		r.Log.Error(err, "unable to connect to kube")
		return err
	}
	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		var tmp *corev1.Pod
		tmp, err = clientset.CoreV1().Pods(pipeline.ObjectMeta.Namespace).Get(r.Ctx, podName, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Errorf("error getting pod %s %w", podName, err)
		}
		if tmp.Status.Phase == corev1.PodRunning {
			r.Log.Info("cockroachdb-0 pod is found to be running!")
			return true, nil
		}
		return false, nil
	})

	err = wait.Poll(time.Duration(1*time.Second), time.Duration(60*time.Second), podCheck)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("timeout waiting for %s to be running...", podName))
		podRunning = false
	} else {
		podRunning = true
	}

	r.Log.Info("cockroachdb-0 pod is deemed running")
	// see if the configmap marker has been created indicating
	// that an init has already been run
	if podRunning {
		var secretFound bool
		var secret corev1.Secret
		err := r.Get(r.Ctx, req.NamespacedName, &secret)
		if err != nil {
			r.Log.Error(err, "unable to get secret for StatefulSets")
		} else {
			secretFound = true
		}

		if !secretFound {
			// run init on database
			// kubectl -n $(PIPELINE) exec -it cockroachdb-0 -- /cockroach/cockroach init --certs-dir=/cockroach/cockroach-certs

			// create init configmap to indicate we have init the database
			var secret corev1.Secret
			secret.ObjectMeta.Labels = map[string]string{"app": "churro", "pipeline": pipeline.ObjectMeta.Name}
			secret.Namespace = pipeline.ObjectMeta.Namespace
			secret.Name = pipeline.ObjectMeta.Name

			if err := ctrl.SetControllerReference(&pipeline, &secret, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(r.Ctx, &secret); err != nil {
				r.Log.Error(err, "unable to create init secret for statefulset", "configmap", secret)
				return err
			}
			r.Log.Info("created init secret for statefulset")
			command := []string{"/cockroach/cockroach", "init", "--certs-dir", "/cockroach/cockroach-certs"}
			//command := []string{"/bin/sh", "ls"}
			clientset, config, err = GetKubeClient()
			containerName := "cockroachdb"
			ns := pipeline.ObjectMeta.Namespace
			pods, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				r.Log.Error(err, "error gettings pods in printPods")
			}
			r.Log.Info(fmt.Sprintf("There are %d pods in the cluster\n", len(pods.Items)))
			a, b, err := r.ExecToPodThroughAPI(config, clientset, command, containerName, podName, ns, nil)
			r.Log.Info(fmt.Sprintf("stdout %s stderr %s\n", a, b))
			if err != nil {
				r.Log.Error(err, "error execing into cockroach for init")
				return err
			}

		}
	}

	podReadyCheck := wait.ConditionFunc(func() (done bool, err error) {
		var tmp *corev1.Pod
		tmp, err = clientset.CoreV1().Pods(pipeline.ObjectMeta.Namespace).Get(r.Ctx, podName, metav1.GetOptions{})
		if err != nil {
			return true, fmt.Errorf("error getting pod %s %w", podName, err)
		}
		for _, v := range tmp.Status.ContainerStatuses {
			if v.Ready {
				r.Log.Info("cockroachdb-0 pod is found to be ready!")
				return true, nil
			}
		}
		return false, nil
	})

	err = wait.Poll(time.Duration(1*time.Second), time.Duration(60*time.Second), podReadyCheck)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("timeout waiting for %s to be ready...", podName))
		return err
	}
	r.Log.Info(fmt.Sprintf("pod %s is ready...", podName))

	return nil
}

// GetKubeClient will get a kubernetes client from the following sources:
// - a path to the kubeconfig file passed on the command line (--kubeconfig)
// - an environment variable that specifies the path (export KUBECONFIG)
// - the user's $HOME/.kube/config file
// - in-cluster connection for when the sdk is run within a cluster instead of
//   the command line
func GetKubeClient() (client *kubernetes.Clientset, config *rest.Config, err error) {

	config, err = ctrl.GetConfig()
	if err != nil {
		return client, config, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, config, err
	}

	return clientset, config, err
}

// ExecToPodThroughAPI uninterractively exec to the pod with the command specified.
// :param string command: list of the str which specify the command.
// :param string pod_name: Pod name
// :param string namespace: namespace of the Pod.
// :param io.Reader stdin: Standerd Input if necessary, otherwise `nil`
// :return: string: Output of the command. (STDOUT)
//          string: Errors. (STDERR)
//           error: If any error has occurred otherwise `nil`
func (r PipelineReconciler) ExecToPodThroughAPI(config *rest.Config, clientset *kubernetes.Clientset, command []string, containerName, podName, namespace string, stdin io.Reader) (string, string, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		r.Log.Error(err, "error adding to scheme")
		return "", "", err
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	r.Log.Info(fmt.Sprintf("Request object: %v", req))
	r.Log.Info(fmt.Sprintf("Request URL object: %v", req.URL()))
	r.Log.Info(fmt.Sprintf("Request URL: %s", req.URL().String()))

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		r.Log.Error(err, "error in remotecommand")
		return "", "", err
	}

	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		r.Log.Error(err, "error in remotecommand stream")
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}
