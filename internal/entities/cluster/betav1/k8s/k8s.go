package k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GaleraCluster is the Schema for the galeraclusters API
type GaleraCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GaleraClusterSpec   `json:"spec,omitempty"`
	Status GaleraClusterStatus `json:"status,omitempty"`
}

// GaleraClusterSpec defines the desired state of GaleraCluster
type GaleraClusterSpec struct {
	// Number of replicas in the Galera cluster
	Replicas int `json:"replicas,omitempty"`

	// Docker image to use for the Galera cluster
	Image string `json:"image,omitempty"`

	// Status of Galera Cluster
	Status int32 `json:"status,omitempty"`
}

// GaleraClusterStatus defines the observed state of GaleraCluster
type GaleraClusterStatus struct {
	// Current status of the Galera cluster
	Status int32 `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GaleraClusterList contains a list of GaleraCluster
type GaleraClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GaleraCluster `json:"items"`
}
