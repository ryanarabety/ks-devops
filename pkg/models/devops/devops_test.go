/*
Copyright 2020 The KubeSphere Authors.

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

package devops

import (
	"context"
	"fmt"
	v12 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
	"kubesphere.io/devops/pkg/client/clientset/versioned"
	fakeclientset "kubesphere.io/devops/pkg/client/clientset/versioned/fake"
	"kubesphere.io/devops/pkg/constants"

	"kubesphere.io/devops/pkg/client/devops"
	"kubesphere.io/devops/pkg/client/devops/fake"
)

const baseUrl = "http://127.0.0.1/kapis/devops.kubesphere.io/v1alpha2/"

func TestGetNodesDetail(t *testing.T) {
	fakeData := make(map[string]interface{})
	PipelineRunNodes := []devops.PipelineRunNodes{
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "1",
			Result:      "SUCCESS",
		},
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "2",
			Result:      "SUCCESS",
		},
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "3",
			Result:      "SUCCESS",
		},
	}

	NodeSteps := []devops.NodeSteps{
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "1",
			Result:      "SUCCESS",
		},
	}

	fakeData["project1-pipeline1-run1"] = PipelineRunNodes
	fakeData["project1-pipeline1-run1-1"] = NodeSteps
	fakeData["project1-pipeline1-run1-2"] = NodeSteps
	fakeData["project1-pipeline1-run1-3"] = NodeSteps

	devopsClient := fake.NewFakeDevops(fakeData)

	devopsOperator := NewDevopsOperator(devopsClient, nil, nil)

	httpReq, _ := http.NewRequest(http.MethodGet, baseUrl+"devops/project1/pipelines/pipeline1/runs/run1/nodesdetail/?limit=10000", nil)

	nodesDetails, err := devopsOperator.GetNodesDetail("project1", "pipeline1", "run1", httpReq)
	if err != nil || nodesDetails == nil {
		t.Fatalf("should not get error %+v", err)
	}

	for _, v := range nodesDetails {
		if v.Steps[0].ID == "" {
			t.Fatalf("Can not get any step.")
		}
	}
}

func TestGetBranchNodesDetail(t *testing.T) {
	fakeData := make(map[string]interface{})

	BranchPipelineRunNodes := []devops.BranchPipelineRunNodes{
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "1",
			Result:      "SUCCESS",
		},
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "2",
			Result:      "SUCCESS",
		},
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "3",
			Result:      "SUCCESS",
		},
	}

	BranchNodeSteps := []devops.NodeSteps{
		{
			DisplayName: "Deploy to Kubernetes",
			ID:          "1",
			Result:      "SUCCESS",
		},
	}

	fakeData["project1-pipeline1-branch1-run1"] = BranchPipelineRunNodes
	fakeData["project1-pipeline1-branch1-run1-1"] = BranchNodeSteps
	fakeData["project1-pipeline1-branch1-run1-2"] = BranchNodeSteps
	fakeData["project1-pipeline1-branch1-run1-3"] = BranchNodeSteps

	devopsClient := fake.NewFakeDevops(fakeData)

	devopsOperator := NewDevopsOperator(devopsClient, nil, nil)

	httpReq, _ := http.NewRequest(http.MethodGet, baseUrl+"devops/project1/pipelines/pipeline1/branchs/branch1/runs/run1/nodesdetail/?limit=10000", nil)

	nodesDetails, err := devopsOperator.GetBranchNodesDetail("project1", "pipeline1", "branch1", "run1", httpReq)
	if err != nil || nodesDetails == nil {
		t.Fatalf("should not get error %+v", err)
	}

	for _, v := range nodesDetails {
		if v.Steps[0].ID == "" {
			t.Fatalf("Can not get any step.")
		}
	}
}

func Test_devopsOperator_CreateDevOpsProject(t *testing.T) {
	type fields struct {
		ksclient versioned.Interface
	}
	type args struct {
		workspace string
		project   *v1alpha3.DevOpsProject
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		verify  func(client versioned.Interface, args args, t *testing.T) bool
		wantErr bool
	}{{
		name: "lack of the generate name",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(),
		},
		args: args{
			workspace: "ws",
			project:   &v1alpha3.DevOpsProject{},
		},
		wantErr: true,
	}, {
		name: "duplicated in the same workspace",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
					Labels: map[string]string{
						constants.WorkspaceLabelKey: "ws",
					},
				},
			}),
		},
		args: args{
			workspace: "ws",
			project: &v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
				},
			},
		},
		wantErr: true,
	}, {
		name: "allow the same name in the different workspaces",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
					Name:         "generated-name",
					Labels: map[string]string{
						constants.WorkspaceLabelKey: "ws1",
					},
				},
			}),
		},
		args: args{
			workspace: "ws",
			project: &v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
				},
			},
		},
		wantErr: false,
	}, {
		name: "normal case",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(),
		},
		args: args{
			workspace: "ws",
			project: &v1alpha3.DevOpsProject{
				ObjectMeta: v1.ObjectMeta{
					GenerateName: "devops",
				},
			},
		},
		wantErr: false,
		verify: func(client versioned.Interface, args args, t *testing.T) bool {
			list, err := client.DevopsV1alpha3().DevOpsProjects().List(context.TODO(), metav1.ListOptions{})
			assert.Nil(t, err)
			assert.Equal(t, len(list.Items) > 0, true)

			for i := range list.Items {
				item := list.Items[i]

				if item.GenerateName == args.project.GenerateName {
					assert.NotNil(t, item.Annotations)
					assert.NotEmpty(t, item.Annotations[v1alpha3.DevOpeProjectSyncStatusAnnoKey])
					assert.NotEmpty(t, item.Annotations[v1alpha3.DevOpeProjectSyncTimeAnnoKey])

					assert.NotNil(t, item.Labels)
					assert.Equal(t, args.workspace, item.Labels[constants.WorkspaceLabelKey])
					return true
				}
			}
			return false
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := devopsOperator{
				ksclient: tt.fields.ksclient,
			}
			got, err := d.CreateDevOpsProject(tt.args.workspace, tt.args.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDevOpsProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.verify != nil && !tt.verify(d.ksclient, tt.args, t) {
				t.Errorf("CreateDevOpsProject() got = %v", got)
			}
		})
	}
}

func Test_devopsOperator_GetDevOpsProject(t *testing.T) {
	type fields struct {
		ksclient versioned.Interface
	}
	type args struct {
		workspace   string
		projectName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		verify func(result *v1alpha3.DevOpsProject, resultErr error, t *testing.T)
	}{{
		name: "normal case",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
					Name:         "generated-name",
					Labels: map[string]string{
						constants.WorkspaceLabelKey: "ws",
					},
				},
			}),
		},
		args: args{
			workspace:   "ws",
			projectName: "fake",
		},
		verify: func(result *v1alpha3.DevOpsProject, resultErr error, t *testing.T) {
			assert.Nil(t, resultErr)
			assert.NotNil(t, result)
			assert.Equal(t, "fake", result.GenerateName)
			assert.Equal(t, "generated-name", result.Name)
		},
	}, {
		name: "cannot find by the same generateName in the different workspaces",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
					Name:         "generated-name",
					Labels: map[string]string{
						constants.WorkspaceLabelKey: "ws1",
					},
				},
			}),
		},
		args: args{
			workspace:   "ws",
			projectName: "fake",
		},
		verify: func(result *v1alpha3.DevOpsProject, resultErr error, t *testing.T) {
			assert.NotNil(t, resultErr)
			assert.Nil(t, result)
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := devopsOperator{
				ksclient: tt.fields.ksclient,
			}
			got, err := d.GetDevOpsProjectByGenerateName(tt.args.workspace, tt.args.projectName)
			tt.verify(got, err, t)
		})
	}
}

func Test_devopsOperator_UpdateJenkinsfile(t *testing.T) {
	pipeline := &v1alpha3.Pipeline{}
	pipeline.SetNamespace("ns")
	pipeline.SetName("fake")
	pipeline.Spec.Pipeline = &v1alpha3.NoScmPipeline{}

	type fields struct {
		devopsClient devops.Interface
		k8sclient    kubernetes.Interface
		ksclient     versioned.Interface
		context      context.Context
	}
	type args struct {
		projectName  string
		pipelineName string
		mode         string
		jenkinsfile  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
		verify  func(t *testing.T, ksclient versioned.Interface)
	}{{
		name: "not found pipeline",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(),
		},
		args: args{
			projectName:  "ns",
			pipelineName: "fake",
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err)
			return true
		},
	}, {
		name: "json mode",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(pipeline.DeepCopy()),
		},
		args: args{
			projectName:  "ns",
			pipelineName: "fake",
			mode:         "json",
			jenkinsfile:  "json-format-jenkinsfile",
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
		verify: func(t *testing.T, ksclient versioned.Interface) {
			pip, err := ksclient.DevopsV1alpha3().Pipelines("ns").Get(context.Background(), "fake", metav1.GetOptions{})
			assert.Nil(t, err)
			assert.Equal(t, "json-format-jenkinsfile", pip.Annotations[v1alpha3.PipelineJenkinsfileValueAnnoKey])
			assert.Equal(t, "json", pip.Annotations[v1alpha3.PipelineJenkinsfileEditModeAnnoKey])
		},
	}, {
		name: "mode value is empty",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(pipeline.DeepCopy()),
		},
		args: args{
			projectName:  "ns",
			pipelineName: "fake",
			mode:         "",
			jenkinsfile:  "jenkinsfile",
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err)
			return true
		},
		verify: func(t *testing.T, ksclient versioned.Interface) {
			pip, err := ksclient.DevopsV1alpha3().Pipelines("ns").Get(context.Background(), "fake", metav1.GetOptions{})
			assert.Nil(t, err)
			assert.Equal(t, "", pip.Spec.Pipeline.Jenkinsfile)
			assert.Equal(t, "", pip.Annotations[v1alpha3.PipelineJenkinsfileEditModeAnnoKey])
		},
	}, {
		name: "mode value is raw",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(pipeline.DeepCopy()),
		},
		args: args{
			projectName:  "ns",
			pipelineName: "fake",
			mode:         "raw",
			jenkinsfile:  "jenkinsfile",
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
		verify: func(t *testing.T, ksclient versioned.Interface) {
			pip, err := ksclient.DevopsV1alpha3().Pipelines("ns").Get(context.Background(), "fake", metav1.GetOptions{})
			assert.Nil(t, err)
			assert.Equal(t, "jenkinsfile", pip.Spec.Pipeline.Jenkinsfile)
			assert.Equal(t, "raw", pip.Annotations[v1alpha3.PipelineJenkinsfileEditModeAnnoKey])
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := devopsOperator{
				devopsClient: tt.fields.devopsClient,
				k8sclient:    tt.fields.k8sclient,
				ksclient:     tt.fields.ksclient,
				context:      tt.fields.context,
			}
			tt.wantErr(t, d.UpdateJenkinsfile(tt.args.projectName, tt.args.pipelineName, tt.args.mode, tt.args.jenkinsfile), fmt.Sprintf("UpdateJenkinsfile(%v, %v, %v, %v)", tt.args.projectName, tt.args.pipelineName, tt.args.mode, tt.args.jenkinsfile))
			if tt.verify != nil {
				tt.verify(t, tt.fields.ksclient)
			}
		})
	}
}

func Test_devopsOperator_UpdatePipelineObj(t *testing.T) {
	pip := &v1alpha3.Pipeline{}
	pip.SetName("fake")
	pip.SetNamespace("ns")

	pipWithJenkinsfile := pip.DeepCopy()
	pipWithJenkinsfile.Spec.Pipeline = &v1alpha3.NoScmPipeline{}

	project := &v1alpha3.DevOpsProject{}
	project.SetName("ns")
	project.Status.AdminNamespace = "ns"

	type fields struct {
		devopsClient devops.Interface
		k8sclient    kubernetes.Interface
		ksclient     versioned.Interface
		context      context.Context
	}
	type args struct {
		projectName string
		pipeline    *v1alpha3.Pipeline
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1alpha3.Pipeline
		wantErr assert.ErrorAssertionFunc
	}{{
		name: "not found project",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(),
		},
		args: args{
			projectName: "ns",
			pipeline:    pip.DeepCopy(),
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err)
			return true
		},
	}, {
		name: "not found pipeline",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(project.DeepCopy()),
		},
		args: args{
			projectName: "ns",
			pipeline:    pip.DeepCopy(),
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err)
			return true
		},
	}, {
		name: "without jenkinsfile",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(project.DeepCopy(), pip.DeepCopy()),
		},
		args: args{
			projectName: "ns",
			pipeline:    pip.DeepCopy(),
		},
		want: pip.DeepCopy(),
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
	}, {
		name: "normal case, with jenkinsfile",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(project.DeepCopy(), pipWithJenkinsfile.DeepCopy()),
		},
		args: args{
			projectName: "ns",
			pipeline:    pipWithJenkinsfile.DeepCopy(),
		},
		want: pipWithJenkinsfile.DeepCopy(),
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := devopsOperator{
				devopsClient: tt.fields.devopsClient,
				k8sclient:    tt.fields.k8sclient,
				ksclient:     tt.fields.ksclient,
				context:      tt.fields.context,
			}
			got, err := d.UpdatePipelineObj(tt.args.projectName, tt.args.pipeline)
			if !tt.wantErr(t, err, fmt.Sprintf("UpdatePipelineObj(%v, %v)", tt.args.projectName, tt.args.pipeline)) {
				return
			}
			assert.Equalf(t, tt.want, got, "UpdatePipelineObj(%v, %v)", tt.args.projectName, tt.args.pipeline)
		})
	}
}

func Test_devopsOperator_GetJenkinsAgentLabels(t *testing.T) {
	cm := &v12.ConfigMap{
		Data: map[string]string{
			"agent.labels": "good,bad",
		},
	}
	cm.SetName("jenkins-agent-config")
	cm.SetNamespace("kubesphere-devops-system")

	type fields struct {
		devopsClient devops.Interface
		k8sclient    kubernetes.Interface
		ksclient     versioned.Interface
		context      context.Context
	}
	tests := []struct {
		name       string
		fields     fields
		wantLabels []string
		wantErr    assert.ErrorAssertionFunc
	}{{
		name: "not found ConfigMap",
		fields: fields{
			k8sclient: k8sfake.NewSimpleClientset(),
		},
		wantLabels: nil,
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err)
			return true
		},
	}, {
		name: "normal",
		fields: fields{
			k8sclient: k8sfake.NewSimpleClientset(cm.DeepCopy()),
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
		wantLabels: []string{"good", "bad"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := devopsOperator{
				devopsClient: tt.fields.devopsClient,
				k8sclient:    tt.fields.k8sclient,
				ksclient:     tt.fields.ksclient,
				context:      tt.fields.context,
			}
			gotLabels, err := d.GetJenkinsAgentLabels()
			if !tt.wantErr(t, err, fmt.Sprintf("GetJenkinsAgentLabels()")) {
				return
			}
			assert.Equalf(t, tt.wantLabels, gotLabels, "GetJenkinsAgentLabels()")
		})
	}
}

func Test_devopsOperator_CheckDevopsName(t *testing.T) {
	type fields struct {
		ksclient versioned.Interface
	}
	type args struct {
		workspace   string
		projectName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		verify func(result map[string]interface{}, resultErr error, t *testing.T)
	}{{
		name: "normal case",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
					Name:         "generated-name",
					Labels: map[string]string{
						constants.WorkspaceLabelKey: "ws",
					},
				},
			}),
		},
		args: args{
			workspace:   "ws",
			projectName: "fake",
		},
		verify: func(result map[string]interface{}, resultErr error, t *testing.T) {
			assert.Nil(t, resultErr)
			assert.NotNil(t, result)
			assert.Equal(t, true, result["exist"])
		},
	}, {
		name: "cannot find by the same generateName in the different workspaces",
		fields: fields{
			ksclient: fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "fake",
					Name:         "generated-name",
					Labels: map[string]string{
						constants.WorkspaceLabelKey: "ws",
					},
				},
			}),
		},
		args: args{
			workspace:   "ws",
			projectName: "fake1",
		},
		verify: func(result map[string]interface{}, resultErr error, t *testing.T) {
			assert.Nil(t, resultErr)
			assert.NotNil(t, result)
			assert.Equal(t, false, result["exist"])
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := devopsOperator{
				ksclient: tt.fields.ksclient,
			}
			got, err := d.CheckDevopsProject(tt.args.workspace, tt.args.projectName)
			tt.verify(got, err, t)
		})
	}
}
