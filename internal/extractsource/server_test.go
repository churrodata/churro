package extractsource

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/churrodata/churro/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func TestCreateExtractPod(t *testing.T) {

	scheme := "csv"
	filePath := "/tmp/foo"
	cfg := v1alpha1.Pipeline{}
	cfg.Namespace = "mynamespace"
	tableName := "mytable"
	extractSourceName := "foo"

	ns := "default"
	os.Setenv("CHURRO_NAMESPACE", ns)

	s := Server{}

	saDef := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "churro",
			Namespace: ns,
		},
	}

	os.Setenv("FAKECLIENT", "true")
	otherClient, err := GetKubeClient("")
	if err != nil {
		t.Fatalf("operator.CreateExtractPod Error: %v", err)
	}

	_, err = otherClient.CoreV1().ServiceAccounts(ns).Create(context.TODO(), saDef, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("operator.CreateExtractPod Error: %v", err)
	}

	err = s.createExtractPod(otherClient, scheme, filePath, cfg, tableName, extractSourceName)
	if err != nil {
		t.Fatalf("operator.CreateExtractPod Error: %v", err)
	}

}

func TestGetExtractCount(t *testing.T) {

	ns := "default"
	os.Setenv("CHURRO_NAMESPACE", ns)

	s := Server{}

	os.Setenv("FAKECLIENT", "true")
	otherClient, err := GetKubeClient("")
	if err != nil {
		t.Fatalf("operator.CreateExtractPod Error: %v", err)
	}

	count, err := getExtractCount(otherClient.(kubernetes.Clientset), ns)
	if err != nil {
		t.Fatalf("operator.CreateExtractPod Error: %v", err)
	}

	fmt.Printf("count is %d\n", count)

}
