package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Endpoint struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Scheme string `json:"scheme"`
}

type Source struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Path      string `json:"path"`
	Port      int    `json:"port"`
	Scheme    string `json:"scheme"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Database  string `json:"database"`
	Tablename string `json:"tablename"`
}

type DBCreds struct {
	CAKey         string `json:"cakey"`
	CACrt         string `json:"cacrt"`
	NodeKey       string `json:"nodekey"`
	NodeCrt       string `json:"nodecrt"`
	ClientRootCrt string `json:"clientrootcrt"`
	ClientRootKey string `json:"clientrootkey"`
	PipelineCrt   string `json:"pipelinecrt"`
	PipelineKey   string `json:"pipelinekey"`
}
type ServiceCreds struct {
	ServiceCrt string `json:"servicecrt"`
	ServiceKey string `json:"servicekey"`
}
type WatchConfigStruct struct {
	Location Endpoint `json:"location"`
}

type Extension struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type ExtractRuleDefinition struct {
	ID                    string `json:"id"`
	Extractsourceid       string `json:"extractsourceid"`
	ColumnName            string `json:"columnname"`
	ColumnPath            string `json:"columnpath"`
	ColumnType            string `json:"columntype"`
	MatchValues           string `json:"matchvalues"`
	TransformFunctionName string `json:"transformfunctionname"`
}

type TransformFunction struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}

type ExtensionDefinition struct {
	ID              string `json:"id"`
	Extractsourceid string `json:"extractsourceid"`
	Extensionname   string `json:"extensionname"`
	Extensionpath   string `json:"extensionpath"`
}

type ExtractSourceDefinition struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	Scheme         string `json:"scheme"`
	Regex          string `json:"regex"`
	Tablename      string `json:"tablename"`
	Cronexpression string `json:"cronexpression"`
	Skipheaders    int    `json:"skipheaders"`
	Multiline      string `json:"multiline"`
	Sheetname      string `json:"sheetname"`
	Port           int    `json:"port"`
	Encoding       string `json:"encoding"`
	Transport      string `json:"transport"`
	Servicetype    string `json:"servicetype"`
}

// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	Id                 string `json:"id"`
	MaxJobs            int    `json:"maxjobs"`
	DatabaseType       string `json:"databasetype"`
	StorageClassName   string `json:"storageclassname"`
	StorageSize        string `json:"storagesize"`
	AccessMode         string `json:"accessmode"`
	HarvestFrequency   string `json:"harvestfrequency"`
	HarvestPodDuration string `json:"harvestpodduration"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Quantity of instances
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	//Size int32 `json:"size,omitempty"`

	// Name of the ConfigMap for GuestbookSpec's configuration
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	//ConfigMapName string `json:"configMapName"`

	// +kubebuilder:validation:Enum=Phone;Address;Name
	Type            string `json:"alias,omitempty"`
	AdminDataSource Source `json:"adminDataSource,omitempty"`
	DataSource      Source `json:"dataSource,omitempty"`

	//WatchDirectories []WatchDirectory `json:"watchDirectories"`
	WatchConfig         WatchConfigStruct         `json:"watchConfig"`
	DatabaseCredentials DBCreds                   `json:"dbcreds,omitempty"`
	ServiceCredentials  ServiceCreds              `json:"servicecreds,omitempty"`
	Functions           []TransformFunction       `json:"functions,omitempty"`
	Extractsources      []ExtractSourceDefinition `json:"extractsources,omitempty"`
	Extensions          []ExtensionDefinition     `json:"extensions,omitempty"`
	Extractrules        []ExtractRuleDefinition   `json:"extractrules,omitempty"`
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PodName of the active Guestbook node.
	Active string `json:"active"`

	// PodNames of the standby Guestbook nodes.
	Standby []string `json:"standby"`
}

// +kubebuilder:object:root=true

// Pipeline is the Schema for the pipelines API
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
	Status PipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
