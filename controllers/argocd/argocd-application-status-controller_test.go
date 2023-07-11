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

package argocd

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"kubesphere.io/devops/controllers/core"
	"kubesphere.io/devops/pkg/api/gitops/v1alpha1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Test_getArgoCDApplication(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	defaultApp := &unstructured.Unstructured{}
	defaultApp.SetKind("Application")
	defaultApp.SetAPIVersion("argoproj.io/v1alpha1")
	defaultApp.SetName("name")
	defaultApp.SetNamespace("ns")

	type args struct {
		client         client.Reader
		namespacedName types.NamespacedName
	}
	tests := []struct {
		name   string
		args   args
		verify func(t *testing.T, app *unstructured.Unstructured, err error)
	}{{
		name: "normal case",
		args: args{
			client: fake.NewFakeClientWithScheme(schema, defaultApp.DeepCopy()),
			namespacedName: types.NamespacedName{
				Namespace: "ns",
				Name:      "name",
			},
		},
		verify: func(t *testing.T, app *unstructured.Unstructured, err error) {
			assert.Nil(t, err)
			assert.Equal(t, "ns", app.GetNamespace())
			assert.Equal(t, "name", app.GetName())
		},
	}, {
		name: "not found",
		args: args{
			client: fake.NewFakeClientWithScheme(schema),
		},
		verify: func(t *testing.T, app *unstructured.Unstructured, err error) {
			assert.NotNil(t, err)
			assert.Nil(t, app)
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApp, err := getArgoCDApplication(tt.args.client, tt.args.namespacedName)
			tt.verify(t, gotApp, err)
		})
	}
}

func Test_getArgoCDApplicationObject(t *testing.T) {
	tests := []struct {
		name   string
		verify func(t *testing.T, obj *unstructured.Unstructured)
	}{{
		name: "normal case",
		verify: func(t *testing.T, obj *unstructured.Unstructured) {
			assert.NotNil(t, obj)

			assert.Equal(t, "Application", obj.GetKind())
			assert.Equal(t, "argoproj.io/v1alpha1", obj.GetAPIVersion())
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.verify(t, createBareArgoCDApplicationObject())
		})
	}
}

func TestArgoCDApplicationStatusReconciler_Reconcile(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)
	err = v1alpha1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	defaultApp := &v1alpha1.Application{}
	defaultApp.SetName("name")
	defaultApp.SetNamespace("ns")
	defaultApp.Spec.ArgoApp = &v1alpha1.ArgoApplication{}

	defaultArgoCDApp := &unstructured.Unstructured{}
	defaultArgoCDApp.SetKind("Application")
	defaultArgoCDApp.SetAPIVersion("argoproj.io/v1alpha1")
	defaultArgoCDApp.SetName("name")
	defaultArgoCDApp.SetNamespace("ns")

	appWithStatus := defaultArgoCDApp.DeepCopy()
	appWithStatus.SetLabels(map[string]string{
		v1alpha1.AppNamespaceLabelKey: "ns",
		v1alpha1.AppNameLabelKey:      "name",
	})
	_ = unstructured.SetNestedMap(appWithStatus.Object, map[string]interface{}{
		"summary": map[string]interface{}{
			"images": []interface{}{"nginx"},
		},
		"sync": map[string]interface{}{
			"status": "ready",
		},
		"health": map[string]interface{}{
			"status": "ready",
		},
	}, "status")

	type fields struct {
		Client client.Client
	}
	type args struct {
		req controllerruntime.Request
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantResult controllerruntime.Result
		wantErr    assert.ErrorAssertionFunc
	}{{
		name: "normal case",
		fields: fields{
			Client: fake.NewFakeClientWithScheme(schema, defaultArgoCDApp, defaultApp),
		},
		args: args{
			req: controllerruntime.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "ns",
					Name:      "name",
				},
			},
		},
		wantResult: controllerruntime.Result{},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return false
		},
	}, {
		name: "not found ArgoCD application",
		fields: fields{
			Client: fake.NewFakeClientWithScheme(schema),
		},
		args: args{
			req: controllerruntime.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "ns",
					Name:      "name",
				},
			},
		},
		wantResult: controllerruntime.Result{},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		},
	}, {
		name: "not found ks application",
		fields: fields{
			Client: fake.NewFakeClientWithScheme(schema, defaultArgoCDApp),
		},
		args: args{
			req: controllerruntime.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "ns",
					Name:      "name",
				},
			},
		},
		wantResult: controllerruntime.Result{},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return true
		},
	}, {
		name: "have status from argo application",
		fields: fields{
			Client: fake.NewFakeClientWithScheme(schema, appWithStatus, defaultApp),
		},
		args: args{
			req: controllerruntime.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "ns",
					Name:      "name",
				},
			},
		},
		wantResult: controllerruntime.Result{},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)

			c := i[1].(client.Client)
			app := &v1alpha1.Application{}
			err = c.Get(context.Background(), types.NamespacedName{
				Namespace: "ns",
				Name:      "name",
			}, app)
			assert.Nil(t, err)

			assert.Equal(t, "nginx", app.Annotations[v1alpha1.AnnoKeyImages])
			return true
		},
	}, {
		name: "cannot found inner app",
		fields: fields{
			Client: fake.NewFakeClientWithScheme(schema, appWithStatus),
		},
		args: args{
			req: controllerruntime.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "ns",
					Name:      "name",
				},
			},
		},
		wantResult: controllerruntime.Result{},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ApplicationStatusReconciler{
				Client:   tt.fields.Client,
				log:      logr.New(log.NullLogSink{}),
				recorder: &record.FakeRecorder{},
			}
			gotResult, err := r.Reconcile(context.Background(), tt.args.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Reconcile(%v)", tt.args.req), tt.fields.Client) {
				return
			}
			assert.Equalf(t, tt.wantResult, gotResult, "Reconcile(%v)", tt.args.req)
		})
	}
}

func Test_parseArgoStatus(t *testing.T) {
	type args struct {
		dataFile string
	}
	tests := []struct {
		name       string
		args       args
		wantStatus *argoStatus
		wantErr    assert.ErrorAssertionFunc
	}{{
		name: "normal",
		args: args{dataFile: "data/argo-status.json"},
		wantStatus: &argoStatus{argoStatusSummary{
			Images: []string{"ghcr.io/linuxsuren-bot/open-podcasts-ui:v1.0.2",
				"ghcr.io/linuxsuren-bot/open-podcasts:v1.0.0",
				"ghcr.io/opensource-f2f/kube-rbac-proxy:v0.8.0",
				"ghcr.io/opensource-f2f/open-podcasts-apiserver:dev"}}},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
	}, {
		name:       "no summary",
		args:       args{dataFile: "data/argo-status-without-summary.json"},
		wantStatus: &argoStatus{},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return true
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ioutil.ReadFile(tt.args.dataFile)
			assert.Nil(t, err)

			gotStatus, err := parseArgoStatus(data)
			if !tt.wantErr(t, err, fmt.Sprintf("parseArgoStatus(%v)", tt.args.dataFile)) {
				return
			}
			assert.Equalf(t, tt.wantStatus, gotStatus, "parseArgoStatus(%v)", tt.args.dataFile)
		})
	}
}

func TestApplicationStatusReconciler_SetupWithManager(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	type fields struct {
		Client   client.Client
		log      logr.Logger
		recorder record.EventRecorder
	}
	type args struct {
		mgr controllerruntime.Manager
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{{
		name: "normal",
		args: args{
			mgr: &core.FakeManager{
				Scheme: schema,
				Client: fake.NewFakeClientWithScheme(schema),
			},
		},
		wantErr: core.NoErrors,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ApplicationStatusReconciler{
				Client:   tt.fields.Client,
				log:      tt.fields.log,
				recorder: tt.fields.recorder,
			}
			tt.wantErr(t, r.SetupWithManager(tt.args.mgr), fmt.Sprintf("SetupWithManager(%v)", tt.args.mgr))
		})
	}
}
