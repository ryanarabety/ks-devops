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

package pipelinerun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/types"
	cmstore "kubesphere.io/devops/pkg/store/configmap"
	"net/url"
	"strconv"

	"kubesphere.io/devops/pkg/kapis"

	"github.com/emicklei/go-restful"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
	"kubesphere.io/devops/pkg/apiserver/query"
	apiserverrequest "kubesphere.io/devops/pkg/apiserver/request"
	"kubesphere.io/devops/pkg/client/devops"
	devopsClient "kubesphere.io/devops/pkg/client/devops"
	"kubesphere.io/devops/pkg/models/pipelinerun"
	resourcesV1alpha3 "kubesphere.io/devops/pkg/models/resources/v1alpha3"
)

// apiHandlerOption holds some useful tools for API handler.
type apiHandlerOption struct {
	devopsClient devopsClient.Interface
	client       client.Client
}

// apiHandler contains functions to handle coming request and give a response.
type apiHandler struct {
	apiHandlerOption
}

// newAPIHandler creates an APIHandler.
func newAPIHandler(o apiHandlerOption) *apiHandler {
	return &apiHandler{o}
}

func (h *apiHandler) listPipelineRuns(request *restful.Request, response *restful.Response) {
	nsName := request.PathParameter("namespace")
	pipName := request.PathParameter("pipeline")
	branchName := request.QueryParameter("branch")
	backward, err := strconv.ParseBool(request.QueryParameter("backward"))
	if err != nil {
		// by default, we have to guarantee backward compatibility
		backward = true
	}

	queryParam := query.ParseQueryParameter(request)

	// validate the Pipeline
	pipeline := &v1alpha3.Pipeline{}
	err = h.client.Get(context.Background(), client.ObjectKey{Namespace: nsName, Name: pipName}, pipeline)
	if err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	// build label selector
	labelSelector, err := buildLabelSelector(queryParam, pipeline.Name)
	if err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	opts := make([]client.ListOption, 0, 3)
	opts = append(opts, client.InNamespace(pipeline.Namespace))
	opts = append(opts, client.MatchingLabelsSelector{Selector: labelSelector})
	if branchName != "" {
		opts = append(opts, client.MatchingFields{v1alpha3.PipelineRunSCMRefNameField: branchName})
	}

	var prs v1alpha3.PipelineRunList
	// fetch PipelineRuns
	if err := h.client.List(context.Background(), &prs, opts...); err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	var listHandler resourcesV1alpha3.ListHandler = listHandler{}
	if backward {
		listHandler = backwardListHandler{}
	}
	apiResult := resourcesV1alpha3.ToListResult(convertPipelineRunsToObject(prs.Items), queryParam, listHandler)
	_ = response.WriteAsJson(apiResult)
}

func (h *apiHandler) createPipelineRun(request *restful.Request, response *restful.Response) {
	nsName := request.PathParameter("namespace")
	pipName := request.PathParameter("pipeline")
	branch := request.QueryParameter("branch")
	payload := devops.RunPayload{}
	if err := request.ReadEntity(&payload); err != nil && err != io.EOF {
		kapis.HandleBadRequest(response, request, err)
		return
	}
	// validate the Pipeline
	var pipeline v1alpha3.Pipeline
	if err := h.client.Get(context.Background(), client.ObjectKey{Namespace: nsName, Name: pipName}, &pipeline); err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	var (
		scm *v1alpha3.SCM
		err error
	)
	if scm, err = CreateScm(&pipeline.Spec, branch); err != nil {
		kapis.HandleBadRequest(response, request, err)
		return
	}

	// get current login user from request context
	user, ok := apiserverrequest.UserFrom(request.Request.Context())
	if !ok || user == nil {
		// should never happen
		err := fmt.Errorf("unauthenticated user entered to create PipelineRun for Pipeline '%s/%s'", nsName, pipName)
		kapis.HandleUnauthorized(response, request, err)
		return
	}
	// create PipelineRun
	pr := CreatePipelineRun(&pipeline, &payload, scm)
	if user.GetName() != "" {
		pr.GetAnnotations()[v1alpha3.PipelineRunCreatorAnnoKey] = user.GetName()
	}
	if err := h.client.Create(context.Background(), pr); err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	_ = response.WriteEntity(pr)
}

func (h *apiHandler) getPipelineRun(request *restful.Request, response *restful.Response) {
	nsName := request.PathParameter("namespace")
	prName := request.PathParameter("pipelinerun")

	// get pipelinerun
	var pr v1alpha3.PipelineRun
	if err := h.client.Get(context.Background(), client.ObjectKey{Namespace: nsName, Name: prName}, &pr); err != nil {
		kapis.HandleError(request, response, err)
		return
	}
	_ = response.WriteEntity(&pr)
}

func (h *apiHandler) getNodeDetails(request *restful.Request, response *restful.Response) {
	namespaceName := request.PathParameter("namespace")
	pipelineRunName := request.PathParameter("pipelinerun")
	ctx := request.Request.Context()

	// get pipelinerun
	pr := &v1alpha3.PipelineRun{}
	if err := h.client.Get(ctx, client.ObjectKey{Namespace: namespaceName, Name: pipelineRunName}, pr); err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	// get stage status
	stagesJSON, ok := pr.Annotations[v1alpha3.JenkinsPipelineRunStagesStatusAnnoKey]
	if !ok {
		if pipelineRunStore, err := cmstore.NewConfigMapStore(ctx, types.NamespacedName{
			Namespace: namespaceName,
			Name:      pipelineRunName,
		}, h.client); err != nil {
			// If the stages status does not exist, set it as an empty array
			stagesJSON = "[]"
		} else {
			stagesJSON = pipelineRunStore.GetStages()
		}
	}

	var stages []pipelinerun.NodeDetail
	if err := json.Unmarshal([]byte(stagesJSON), &stages); err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	// TODO(johnniang): Check current user Handle the approvable field of NodeDetail
	// this is a temporary solution of approvable
	for i := range stages {
		for j := range stages[i].Steps {
			stages[i].Steps[j].Approvable = true
		}
	}

	_ = response.WriteEntity(&stages)
}

// downloadArtifact API to download artifacts from Jenkins
func (h *apiHandler) downloadArtifact(request *restful.Request, response *restful.Response) {
	namespaceName := request.PathParameter("namespace")
	pipelineRunName := request.PathParameter("pipelinerun")
	filename := request.QueryParameter("filename")

	// get pipelinerun
	pr := &v1alpha3.PipelineRun{}
	err := h.client.Get(context.Background(), client.ObjectKey{Namespace: namespaceName, Name: pipelineRunName}, pr)
	if err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	filename, err = url.QueryUnescape(filename)
	if err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	buildID, exists := pr.GetPipelineRunID()
	if !exists {
		kapis.HandleError(request, response, fmt.Errorf("unable to get PipelineRun nodes due to not found run ID"))
		return
	}
	pipelineName := pr.Labels[v1alpha3.PipelineNameLabelKey]
	isMultiBranch := pr.Spec.IsMultiBranchPipeline()
	branchName := pr.GetRefName()

	// request the Jenkins API to download artifact
	body, err := h.devopsClient.DownloadArtifact(namespaceName, pipelineName, buildID, filename, isMultiBranch, branchName)
	if err != nil {
		kapis.HandleError(request, response, err)
		return
	}
	defer func() {
		_ = body.Close()
	}()

	buf := &bytes.Buffer{}
	if _, err = io.Copy(buf, body); err != nil {
		kapis.HandleError(request, response, err)
		return
	}

	// add download header
	response.AddHeader("Content-Type", "application/octet-stream")
	response.AddHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	_, err = response.Write(buf.Bytes())
	if err != nil {
		kapis.HandleError(request, response, err)
		return
	}
}
