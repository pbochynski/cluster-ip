/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterIPSpec defines the desired state of ClusterIP
type ClusterIPSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//+kubebuilder:default=topology.kubernetes.io/zone
	NodeSpreadLabel string `json:"nodeSpreadLabel,omitempty"`
}
type NodeIP struct {
	NodeLabel      string      `json:"nodeLabel"`
	IP             string      `json:"ip"`
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// ClusterIPStatus defines the observed state of ClusterIP
type ClusterIPStatus struct {
	// State signifies current state of Module CR.
	// Value can be one of ("Ready", "Processing", "Error", "Deleting").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
	State   string   `json:"state"`
	Info    string   `json:"info,omitempty"`
	NodeIPs []NodeIP `json:"nodeIPs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterIP is the Schema for the clusterips API
type ClusterIP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	//+kubebuilder:default={nodeSpreadLabel:"topology.kubernetes.io/zone"}
	Spec   ClusterIPSpec   `json:"spec,omitempty"`
	Status ClusterIPStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterIPList contains a list of ClusterIP
type ClusterIPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterIP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterIP{}, &ClusterIPList{})
}
