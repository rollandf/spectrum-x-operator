/*
 Copyright 2025, NVIDIA CORPORATION & AFFILIATES

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

// SpectrumXConfigSpec defines the desired state of SpectrumXConfig
type SpectrumXConfigSpec struct {
	SpectrumXEWNetworks []RailNetwork    `json:"spectrumXEWNetworks"`
	NetworkMappings     []NetworkMapping `json:"networkMappings"`
	Hosts               []Host           `json:"hosts"`
}

type RailNetwork struct {
	Name            string `json:"name"`
	RailSubnet      string `json:"railSubnet"`
	CrossRailSubnet string `json:"crossRailSubnet"`
}

type NetworkMapping struct {
	Rail      string `json:"rail"`
	Interface string `json:"interface"`
}

type Host struct {
	HostID string     `json:"hostID"`
	Rails  []HostRail `json:"rails"`
}

type HostRail struct {
	RailName       string `json:"railName"`
	RailSubnet     string `json:"railSubnet"`
	PeerLeafPortIP string `json:"peerLeafPortIP"`
}

// SpectrumXConfigStatus defines the observed state of SpectrumXConfig
type SpectrumXConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpectrumXConfig is the Schema for the spectrumxconfigs API
type SpectrumXConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpectrumXConfigSpec   `json:"spec,omitempty"`
	Status SpectrumXConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpectrumXConfigList contains a list of SpectrumXConfig
type SpectrumXConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpectrumXConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpectrumXConfig{}, &SpectrumXConfigList{})
}
