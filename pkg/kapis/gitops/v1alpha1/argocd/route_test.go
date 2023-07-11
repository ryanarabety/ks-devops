// Copyright 2022 KubeSphere Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package argocd

import (
	"bytes"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"kubesphere.io/devops/pkg/api"
	"kubesphere.io/devops/pkg/api/gitops/v1alpha1"
	"kubesphere.io/devops/pkg/apiserver/runtime"
	"kubesphere.io/devops/pkg/config"
	"kubesphere.io/devops/pkg/kapis/common"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestRegisterRoutes(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)

	type args struct {
		service *restful.WebService
		options *common.Options
	}
	tests := []struct {
		name   string
		args   args
		verify func(t *testing.T, service *restful.WebService)
	}{{
		name: "normal case",
		args: args{
			service: runtime.NewWebService(v1alpha1.GroupVersion),
			options: &common.Options{GenericClient: fake.NewFakeClientWithScheme(schema)},
		},
		verify: func(t *testing.T, service *restful.WebService) {
			assert.Greater(t, len(service.Routes()), 0)
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterRoutes(tt.args.service, tt.args.options, &config.ArgoCDOption{})
			tt.verify(t, tt.args.service)
		})
	}
}

func TestArgoAPIs(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	app := v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "app",
			Labels: map[string]string{
				v1alpha1.HealthStatusLabelKey: "Healthy",
				v1alpha1.SyncStatusLabelKey:   "Synced",
			},
		},
	}

	nonArgoClusterSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-argo-cluster",
			Namespace: "ns",
		},
	}
	invalidArgoClusterSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-argo-cluster",
			Namespace: "ns",
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "cluster",
			},
		},
		Data: map[string][]byte{
			"server": []byte("server"),
		},
	}
	validArgoClusterSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argo-cluster",
			Namespace: "ns",
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "cluster",
			},
		},
		Data: map[string][]byte{
			"name":   []byte("name"),
			"server": []byte("server"),
		},
	}

	type request struct {
		method string
		uri    string
		body   func() io.Reader
	}
	tests := []struct {
		name         string
		request      request
		responseCode int
		k8sclient    client.Client
		verify       func(t *testing.T, body []byte)
	}{{
		name: "get clusters, no expected data",
		request: request{
			method: http.MethodGet,
			uri:    "/clusters",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, nonArgoClusterSecret.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			assert.Equal(t, `[
 {
  "server": "https://kubernetes.default.svc",
  "name": "in-cluster"
 }
]`, string(body))
		},
	}, {
		name: "get clusters, have invalid data",
		request: request{
			method: http.MethodGet,
			uri:    "/clusters",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, invalidArgoClusterSecret.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			assert.Equal(t, `[
 {
  "server": "https://kubernetes.default.svc",
  "name": "in-cluster"
 }
]`, string(body))
		},
	}, {
		name: "get clusters, have the expected data",
		request: request{
			method: http.MethodGet,
			uri:    "/clusters",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, validArgoClusterSecret.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			assert.Equal(t, `[
 {
  "server": "https://kubernetes.default.svc",
  "name": "in-cluster"
 },
 {
  "server": "server",
  "name": "name"
 }
]`, string(body))
		},
	}, {
		name: "get applications summary",
		request: request{
			method: http.MethodGet,
			uri:    "/namespaces/ns/application-summary",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, app.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			assert.JSONEq(t, `{"total": 1, "healthStatus": { "Healthy": 1 }, "syncStatus": { "Synced": 1 }}`, string(body))
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsWithGroup := runtime.NewWebService(v1alpha1.GroupVersion)
			RegisterRoutes(wsWithGroup, &common.Options{GenericClient: tt.k8sclient}, &config.ArgoCDOption{})

			container := restful.NewContainer()
			container.Add(wsWithGroup)

			api := fmt.Sprintf("http://fake.com/kapis/gitops.kubesphere.io/%s%s", v1alpha1.GroupVersion.Version, tt.request.uri)
			var body io.Reader
			if tt.request.body != nil {
				body = tt.request.body()
			}
			req, err := http.NewRequest(tt.request.method, api, body)
			req.Header.Set("Content-Type", "application/json")
			assert.Nil(t, err)

			httpWriter := httptest.NewRecorder()
			container.Dispatch(httpWriter, req)
			assert.Equal(t, tt.responseCode, httpWriter.Code)

			if tt.verify != nil {
				tt.verify(t, httpWriter.Body.Bytes())
			}
		})
	}
}

func TestPublicAPIs(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	argoApp := v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "app",
			Labels: map[string]string{
				v1alpha1.HealthStatusLabelKey: "Healthy",
				v1alpha1.SyncStatusLabelKey:   "Synced",
			},
		},
		Spec: v1alpha1.ApplicationSpec{
			Kind: v1alpha1.ArgoCD,
			ArgoApp: &v1alpha1.ArgoApplication{
				Operation: nil,
				Spec: v1alpha1.ArgoApplicationSpec{
					SyncPolicy: &v1alpha1.SyncPolicy{
						Automated: nil,
					},
				},
			},
		},
	}

	type request struct {
		method string
		uri    string
		body   func() io.Reader
	}
	tests := []struct {
		name         string
		request      request
		responseCode int
		k8sclient    client.Client
		verify       func(t *testing.T, body []byte)
	}{{
		name: "get an empty list of the applications",
		request: request{
			method: http.MethodGet,
			uri:    "/namespaces/fake/applications",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			list := &api.ListResult{}
			err := yaml.Unmarshal(body, list)
			assert.Nil(t, err)

			assert.Equal(t, 0, len(list.Items))
		},
	}, {
		name: "get a normal list of the argo applications",
		request: request{
			method: http.MethodGet,
			uri:    "/namespaces/ns/applications",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, argoApp.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			list := &api.ListResult{}
			err := yaml.Unmarshal(body, list)
			assert.Nil(t, err)

			assert.Equal(t, 1, len(list.Items))
			assert.Nil(t, err)
		},
	}, {
		name: "get a normal argo application",
		request: request{
			method: http.MethodGet,
			uri:    "/namespaces/ns/applications/app",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, argoApp.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			list := &unstructured.Unstructured{}
			err := yaml.Unmarshal(body, list)
			assert.Nil(t, err)

			name, _, err := unstructured.NestedString(list.Object, "metadata", "name")
			assert.Equal(t, "app", name)
			assert.Nil(t, err)
		},
	}, {
		name: "delete an argo application",
		request: request{
			method: http.MethodDelete,
			uri:    "/namespaces/ns/applications/app",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, argoApp.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			list := &unstructured.Unstructured{}
			err := yaml.Unmarshal(body, list)
			assert.Nil(t, err)

			name, _, err := unstructured.NestedString(list.Object, "metadata", "name")
			assert.Equal(t, "app", name)
			assert.Nil(t, err)
		},
	}, {
		name: "delete an argo application by cascade",
		request: request{
			method: http.MethodDelete,
			uri:    "/namespaces/ns/applications/app?cascade=true",
		},
		k8sclient:    fake.NewFakeClientWithScheme(schema, argoApp.DeepCopy()),
		responseCode: http.StatusOK,
		verify: func(t *testing.T, body []byte) {
			list := &unstructured.Unstructured{}
			err := yaml.Unmarshal(body, list)
			assert.Nil(t, err)

			name, _, err := unstructured.NestedString(list.Object, "metadata", "name")
			assert.Equal(t, "app", name)
			finalizers, _, err := unstructured.NestedSlice(list.Object, "metadata", "finalizers")
			assert.Equal(t, []interface{}{"resources-finalizer.argocd.argoproj.io"}, finalizers)
			assert.Nil(t, err)
		},
	},
		{
			name: "create an argo application",
			request: request{
				method: http.MethodPost,
				uri:    "/namespaces/ns/applications",
				body: func() io.Reader {
					return bytes.NewBuffer([]byte(`{
  "apiVersion": "devops.kubesphere.io/v1alpha1",
  "kind": "Application",
  "metadata": {
    "name": "fake"
  },
  "spec": {
	"kind": "argocd",
    "argoApp": {
      "spec": {
        "project": "default"
      }
    }
  }
}`))
				},
			},
			k8sclient:    fake.NewFakeClientWithScheme(schema),
			responseCode: http.StatusOK,
			verify: func(t *testing.T, body []byte) {
				list := &unstructured.Unstructured{}
				err := yaml.Unmarshal(body, list)
				assert.Nil(t, err)

				name, _, err := unstructured.NestedString(list.Object, "metadata", "name")
				assert.Equal(t, "fake", name)
				assert.Nil(t, err)
			},
		},
		{
			name: "create an argo application, invalid payload",
			request: request{
				method: http.MethodPost,
				uri:    "/namespaces/ns/applications",
				body: func() io.Reader {
					return bytes.NewBuffer([]byte(`fake`))
				},
			},
			k8sclient:    fake.NewFakeClientWithScheme(schema),
			responseCode: http.StatusInternalServerError,
		}, {
			name: "update an argo application",
			request: request{
				method: http.MethodPut,
				uri:    "/namespaces/ns/applications/app",
				body: func() io.Reader {
					return bytes.NewBuffer([]byte(`{
  "apiVersion": "devops.kubesphere.io/v1alpha1",
  "kind": "Application",
  "metadata": {
    "name": "app",
    "namespace": "ns"
  },
  "spec": {
	"kind": "argocd",
    "argoApp": {
      "spec": {
        "project": "good"
      }
    }
  }
}`))
				},
			},
			k8sclient:    fake.NewFakeClientWithScheme(schema, argoApp.DeepCopy()),
			responseCode: http.StatusOK,
			verify: func(t *testing.T, body []byte) {
				list := &unstructured.Unstructured{}
				err := yaml.Unmarshal(body, list)
				assert.Nil(t, err)

				project, _, err := unstructured.NestedString(list.Object, "spec", "argoApp", "spec", "project")
				assert.Equal(t, "good", project)
				assert.Nil(t, err)
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsWithGroup := runtime.NewWebService(v1alpha1.GroupVersion)
			RegisterRoutes(wsWithGroup, &common.Options{GenericClient: tt.k8sclient}, &config.ArgoCDOption{
				Enabled:   true,
				Namespace: "argocd",
			})

			container := restful.NewContainer()
			container.Add(wsWithGroup)

			api := fmt.Sprintf("http://fake.com/kapis/gitops.kubesphere.io/%s%s", v1alpha1.GroupVersion.Version, tt.request.uri)
			var body io.Reader
			if tt.request.body != nil {
				body = tt.request.body()
			}
			req, err := http.NewRequest(tt.request.method, api, body)
			req.Header.Set("Content-Type", "application/json")
			assert.Nil(t, err)

			httpWriter := httptest.NewRecorder()
			container.Dispatch(httpWriter, req)
			assert.Equal(t, tt.responseCode, httpWriter.Code)

			if tt.verify != nil {
				tt.verify(t, httpWriter.Body.Bytes())
			}
		})
	}
}
