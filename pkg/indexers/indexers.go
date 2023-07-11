/*
Copyright 2022 The KubeSphere Authors.

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

package indexers

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// CreatePipelineRunSCMRefNameIndexer creates field indexer which could speed up listing PipelineRun by SCM reference name.
func CreatePipelineRunSCMRefNameIndexer(runtimeCache cache.Cache) error {
	return runtimeCache.IndexField(context.Background(),
		&v1alpha3.PipelineRun{},
		v1alpha3.PipelineRunSCMRefNameField,
		extractSCMFunc)
}

func extractSCMFunc(o client.Object) []string {
	pipelineRun, ok := o.(*v1alpha3.PipelineRun)
	if !ok || pipelineRun == nil {
		return []string{}
	}
	if pipelineRun.Spec.SCM == nil {
		return []string{}
	}
	return []string{pipelineRun.Spec.SCM.RefName}
}

// CreatePipelineRunIdentityIndexer creates an indexer which aims for locating a PipelineRun with an identifier, like Pipeline name, SCM reference name and run ID.
func CreatePipelineRunIdentityIndexer(runtimeCache cache.Cache) error {
	// TODO Make the definition of index name in only one place
	return runtimeCache.IndexField(context.Background(),
		&v1alpha3.PipelineRun{},
		v1alpha3.PipelineRunIdentifierIndexerName,
		extractPipelineRunIdentifier)
}

func extractPipelineRunIdentifier(o client.Object) []string {
	pipelineRun, ok := o.(*v1alpha3.PipelineRun)
	if !ok || pipelineRun == nil {
		return []string{}
	}
	return []string{pipelineRun.GetPipelineRunIdentifier()}
}
