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

package git

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubesphere.io/devops/pkg/api/devops/v1alpha1"
	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetClient(t *testing.T) {
	schema, err := v1alpha1.SchemeBuilder.Register().Build()
	assert.Nil(t, err)
	err = v1.SchemeBuilder.AddToScheme(schema)
	assert.Nil(t, err)

	basicSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basicSecret",
			Namespace: "ns",
		},
		Type: v1.SecretTypeBasicAuth,
		Data: map[string][]byte{
			v1.BasicAuthPasswordKey: []byte("token"),
		},
	}
	opaqueSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opaqueSecret",
			Namespace: "ns",
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			v1.ServiceAccountTokenKey: []byte("token"),
		},
	}
	textSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opaqueSecret",
			Namespace: "ns",
		},
		Type: "credential.devops.kubesphere.io/secret-text",
		Data: map[string][]byte{
			v1.ServiceAccountTokenKey: []byte("token"),
		},
	}
	ksBasicAuthSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opaqueSecret",
			Namespace: "ns",
		},
		Type: "credential.devops.kubesphere.io/basic-auth",
		Data: map[string][]byte{
			v1.BasicAuthPasswordKey: []byte("token"),
		},
	}
	type fields struct {
		provider  string
		secretRef *v1.SecretReference
		k8sClient client.Client
		server    string
	}
	type args struct {
		repo *v1alpha3.GitRepository
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantClient *scm.Client
		wantErr    assert.ErrorAssertionFunc
	}{{
		name: "not support git provider",
		fields: fields{
			provider: "not-support",
		},
		args: args{
			repo: &v1alpha3.GitRepository{
				Spec: v1alpha3.GitRepositorySpec{
					Provider: "not-support",
				},
			},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err, i)
			assert.Equal(t, strings.HasPrefix(err.Error(), "Unsupported"), true, i)
			return true
		},
	}, {
		name: "no secret found",
		fields: fields{
			k8sClient: fake.NewFakeClientWithScheme(schema),
			provider:  "github",
			secretRef: &v1.SecretReference{Namespace: "fake", Name: "fake"},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.NotNil(t, err, i)
			return true
		},
	}, {
		name: "github provider",
		fields: fields{
			k8sClient: fake.NewFakeClientWithScheme(schema, basicSecret.DeepCopy()),
			provider:  "github",
			secretRef: &v1.SecretReference{Namespace: "ns", Name: "basicSecret"},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err, i)
			return false
		},
	}, {
		name: "gitlab provider",
		fields: fields{
			k8sClient: fake.NewFakeClientWithScheme(schema, opaqueSecret.DeepCopy()),
			provider:  "gitlab",
			secretRef: &v1.SecretReference{Namespace: "ns", Name: "opaqueSecret"},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err, i)
			return false
		},
	}, {
		name: "gitlab provider - text secret",
		fields: fields{
			k8sClient: fake.NewFakeClientWithScheme(schema, textSecret.DeepCopy()),
			provider:  "gitlab",
			secretRef: &v1.SecretReference{Namespace: "ns", Name: "opaqueSecret"},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err, i)
			return false
		},
	}, {
		name: "gitlab provider - ks basic auth secret",
		fields: fields{
			k8sClient: fake.NewFakeClientWithScheme(schema, ksBasicAuthSecret.DeepCopy()),
			provider:  "gitlab",
			secretRef: &v1.SecretReference{Namespace: "ns", Name: "opaqueSecret"},
		},
		wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err, i)
			return false
		},
	}, {
		name: "bitbucket_cloud",
		fields: fields{
			provider: "bitbucket_cloud",
		},
		wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return false
		},
	}, {
		name: "bitbucket-server",
		fields: fields{
			provider: "bitbucket-server",
			server:   "https://api.bitbucket.org",
		},
		wantErr: func(tt assert.TestingT, err error, i ...interface{}) bool {
			assert.Nil(t, err)
			return false
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewClientFactory(tt.fields.provider, tt.fields.secretRef, tt.fields.k8sClient)
			if tt.fields.server != "" {
				r.Server = tt.fields.server
			}
			gotClient, err := r.GetClient()
			if !tt.wantErr(t, err, fmt.Sprintf("GetClient() %s", tt.name)) {
				return
			}
			assert.Equalf(t, tt.wantClient, gotClient, fmt.Sprintf("GetClient() %s", tt.name))
		})
	}
}
