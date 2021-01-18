/*


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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HostInfoSpec defines the desired state of HostInfo
type HostInfoSpec struct {
	Hostname string `json:"hostname"`
}

// HostInfoStatus defines the observed state of HostInfo
type HostInfoStatus struct {
	TotalCPU int64 `json:"totalCPU"`
	FreeCPU  int64 `json:"freeCPU"`
}

// +kubebuilder:validation:Optional
// +kubebuilder:resource:shortName={"hi"}
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.spec.hostname`
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HostInfo is the Schema for the hostinfoes API
type HostInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostInfoSpec   `json:"spec,omitempty"`
	Status HostInfoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HostInfoList contains a list of HostInfo
type HostInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HostInfo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HostInfo{}, &HostInfoList{})
}
