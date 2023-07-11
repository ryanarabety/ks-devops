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

package v1alpha3

import (
	"context"
	"github.com/jenkins-zh/jenkins-client/pkg/core"
	"kubesphere.io/devops/pkg/jwt/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"kubesphere.io/devops/pkg/api/devops/v1alpha1"
	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
	fakeclientset "kubesphere.io/devops/pkg/client/clientset/versioned/fake"
	fakedevops "kubesphere.io/devops/pkg/client/devops/fake"
	"kubesphere.io/devops/pkg/client/k8s"
	"kubesphere.io/devops/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAPIsExist(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)

	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	container := restful.NewContainer()
	AddToContainer(container, fakedevops.NewFakeDevops(nil), k8s.NewFakeClientSets(k8sfake.NewSimpleClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake", Namespace: "fake",
		},
	}), nil, nil, "", nil,
		fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
			ObjectMeta: metav1.ObjectMeta{Name: "fake"},
			Status:     v1alpha3.DevOpsProjectStatus{AdminNamespace: "fake"},
		}, &v1alpha3.Pipeline{
			ObjectMeta: metav1.ObjectMeta{Namespace: "fake", Name: "fake"},
		})), fake.NewFakeClientWithScheme(schema, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake", Namespace: "fake",
		},
	}), &token.FakeIssuer{}, core.JenkinsCore{})

	type args struct {
		method string
		uri    string
	}
	tests := []struct {
		name       string
		args       args
		body       string
		expectCode int
	}{{
		name: "credential list",
		args: args{
			method: http.MethodGet,
			uri:    "/devops/fake/credentials",
		},
	}, {
		name: "create a credential",
		args: args{
			method: http.MethodPost,
			uri:    "/devops/fake/credentials",
		},
		expectCode: 400,
	}, {
		name: "create a credential with body",
		args: args{
			method: http.MethodPost,
			uri:    "/devops/fake/credentials",
		},
		body:       `{}`,
		expectCode: 200,
	}, {
		name: "get a credential",
		args: args{
			method: http.MethodGet,
			uri:    "/devops/fake/credentials/fake",
		},
	}, {
		name: "update a credential",
		args: args{
			method: http.MethodPut,
			uri:    "/devops/fake/credentials/fake",
		},
		expectCode: 400,
	}, {
		name: "update a credential with body",
		args: args{
			method: http.MethodPut,
			uri:    "/devops/fake/credentials/fake",
		},
		body:       `{}`,
		expectCode: 200,
	}, {
		name: "delete a credential",
		args: args{
			method: http.MethodDelete,
			uri:    "/devops/fake/credentials/fake",
		},
	}, {
		name: "get pipeline list",
		args: args{
			method: http.MethodGet,
			uri:    "/devops/fake/pipelines",
		},
	}, {
		name: "create a pipeline",
		args: args{
			method: http.MethodPost,
			uri:    "/devops/fake/pipelines",
		},
		expectCode: 400,
	}, {
		name: "create a pipeline with body",
		args: args{
			method: http.MethodPost,
			uri:    "/devops/fake/pipelines",
		},
		body:       `{}`,
		expectCode: 200,
	}, {
		name: "get a pipeline",
		args: args{
			method: http.MethodGet,
			uri:    "/devops/fake/pipelines/fake",
		},
	}, {
		name: "update a pipeline without body",
		args: args{
			method: http.MethodPut,
			uri:    "/devops/fake/pipelines/fake",
		},
		expectCode: 400,
	}, {
		name: "update a pipeline with body",
		args: args{
			method: http.MethodPut,
			uri:    "/devops/fake/pipelines/fake",
		},
		body:       `{}`,
		expectCode: 200,
	}, {
		name: "delete a pipeline",
		args: args{
			method: http.MethodDelete,
			uri:    "/devops/fake/pipelines/fake",
		},
	}, {
		name: "get devops list",
		args: args{
			method: http.MethodGet,
			uri:    "/workspaces/fake/devops",
		},
	}, {
		name: "create a devops",
		args: args{
			method: http.MethodPost,
			uri:    "/workspaces/fake/devops",
		},
		expectCode: 400,
	}, {
		name: "create a devops with body",
		args: args{
			method: http.MethodPost,
			uri:    "/workspaces/fake/devops",
		},
		body:       `{}`,
		expectCode: 400,
	}, {
		name: "get a devops",
		args: args{
			method: http.MethodGet,
			uri:    "/workspaces/fake/devops/fake",
		},
	}, {
		name: "update a devops",
		args: args{
			method: http.MethodPut,
			uri:    "/workspaces/fake/devops/fake",
		},
		expectCode: 400,
	}, {
		name: "update a devops with body",
		args: args{
			method: http.MethodPut,
			uri:    "/workspaces/fake/devops/fake",
		},
		body:       `{}`,
		expectCode: 404,
	}, {
		name: "delete a devops",
		args: args{
			method: http.MethodDelete,
			uri:    "/workspaces/fake/devops/fake",
		},
	}, {
		name: "update jenkinsfile without body",
		args: args{
			method: http.MethodPut,
			uri:    "/devops/fake/pipelines/fake-pipeline/jenkinsfile",
		},
		expectCode: 400,
	}, {
		name: "update jenkinsfile",
		args: args{
			method: http.MethodPut,
			uri:    "/devops/fake/pipelines/fake-pipeline/jenkinsfile",
		},
		body:       `{"data":"fake-jenkinsfile"}`,
		expectCode: 404,
	}, {
		name: "get Jenkins labels",
		args: args{
			method: http.MethodGet,
			uri:    "/ci/nodelabels",
		},
		expectCode: http.StatusBadRequest,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpRequest, _ := http.NewRequest(tt.args.method,
				"http://fake.com/kapis/devops.kubesphere.io/v1alpha3"+tt.args.uri, strings.NewReader(tt.body))
			httpRequest = httpRequest.WithContext(context.WithValue(context.TODO(), constants.K8SToken, constants.ContextKeyK8SToken("")))
			httpRequest.Header.Set("Content-Type", "application/json")

			if tt.expectCode == 0 {
				tt.expectCode = http.StatusOK
			}

			httpWriter := httptest.NewRecorder()
			container.Dispatch(httpWriter, httpRequest)
			assert.Equal(t, tt.expectCode, httpWriter.Code)
		})
	}
}

func TestGetDevOpsProject(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	container := restful.NewContainer()

	AddToContainer(container, fakedevops.NewFakeDevops(nil), k8s.NewFakeClientSets(k8sfake.NewSimpleClientset(), nil, nil, "", nil,
		fakeclientset.NewSimpleClientset(&v1alpha3.DevOpsProject{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "fake",
				Name:         "generated-fake",
				Labels: map[string]string{
					constants.WorkspaceLabelKey: "ws",
				},
			},
		})), fake.NewFakeClientWithScheme(schema), &token.FakeIssuer{}, core.JenkinsCore{})

	type args struct {
		method string
		uri    string
	}
	tests := []struct {
		name       string
		args       args
		expectCode int
	}{{
		name: "normal case",
		args: args{
			method: http.MethodGet,
			uri:    "/workspaces/ws/devops/generated-fake",
		},
		expectCode: http.StatusOK,
	}, {
		name: "find by a generateName",
		args: args{
			method: http.MethodGet,
			uri:    "/workspaces/ws/devops/fake?generateName=true",
		},
		expectCode: http.StatusOK,
	}, {
		name: "wrong workspace name",
		args: args{
			method: http.MethodGet,
			uri:    "/workspaces/fake/devops/fake?generateName=true",
		},
		expectCode: http.StatusBadRequest,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpRequest, _ := http.NewRequest(tt.args.method,
				"http://fake.com/kapis/devops.kubesphere.io/v1alpha3"+tt.args.uri, nil)
			httpRequest = httpRequest.WithContext(context.WithValue(context.TODO(), constants.K8SToken, constants.ContextKeyK8SToken("")))

			httpWriter := httptest.NewRecorder()
			container.Dispatch(httpWriter, httpRequest)
			assert.Equal(t, tt.expectCode, httpWriter.Code)
		})
	}
}
