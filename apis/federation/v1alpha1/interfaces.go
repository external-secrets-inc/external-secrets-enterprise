// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package v1alpha1

import "context"

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

type FederationProvider interface {
	GetJWKS(ctx context.Context, token, issuer string, caCrt []byte) (map[string]map[string]string, error)
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil

type ValidationResult string

const (
	ValidationResultValid   ValidationResult = "Valid"
	ValidationResultInvalid ValidationResult = "Invalid"
	ValidationResultUnknown ValidationResult = "Unknown"
	ValidationResultError   ValidationResult = "Error"
)
