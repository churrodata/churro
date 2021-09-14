package uiv1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ChurrouiSpec struct {
	DatabaseType     string `json:"databasetype"`
	ServiceType      string `json:"servicetype"`
	StorageClassName string `json:"storageclassname"`
	StorageSize      string `json:"storagesize"`
	AccessMode       string `json:"accessmode"`
}

type ChurrouiStatus struct {
	Active  string   `json:"active"`
	Standby []string `json:"standby"`
}

// +kubebuilder:object:root=true

// Pipeline is the Schema for the pipelines API
type Churroui struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChurrouiSpec   `json:"spec,omitempty"`
	Status ChurrouiStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ChurrouiList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Churroui `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Churroui{}, &ChurrouiList{})
}
