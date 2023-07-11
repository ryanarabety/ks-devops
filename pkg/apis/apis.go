/*
Copyright 2019 The KubeSphere Authors.

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

// Package apis contains KubeSphere API groups.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
	"kubesphere.io/devops/pkg/api/gitops/v1alpha1"
	helmv2 "kubesphere.io/devops/pkg/external/fluxcd/helm/v2beta1"
	kusv1 "kubesphere.io/devops/pkg/external/fluxcd/kustomize/v1beta2"
	sourcev1 "kubesphere.io/devops/pkg/external/fluxcd/source/v1beta2"
)

// addToSchemes may be used to add all resources defined in the project to a Scheme
var addToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) {
	utilruntime.Must(addToSchemes.AddToScheme(s))
}

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	addToSchemes = append(addToSchemes,
		v1alpha3.SchemeBuilder.AddToScheme,
		v1alpha1.SchemeBuilder.AddToScheme,
		helmv2.SchemeBuilder.AddToScheme,
		kusv1.SchemeBuilder.AddToScheme,
		sourcev1.SchemeBuilder.AddToScheme)
}
