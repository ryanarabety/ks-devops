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

package k8s

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"reflect"
	"testing"
)

func TestNewKubernetesClientWithConfig(t *testing.T) {
	type args struct {
		config *rest.Config
	}
	tests := []struct {
		name       string
		args       args
		wantClient Client
		wantErr    bool
	}{{
		name:       "nil arg",
		args:       args{},
		wantClient: nil,
		wantErr:    false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClient, err := NewKubernetesClientWithConfig(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKubernetesClientWithConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotClient, tt.wantClient) {
				t.Errorf("NewKubernetesClientWithConfig() gotClient = %v, want %v", gotClient, tt.wantClient)
			}
		})
	}
}

func TestNewKubernetesClientWithToken(t *testing.T) {
	type args struct {
		token  string
		master string
	}
	tests := []struct {
		name       string
		args       args
		wantClient Client
		wantErr    bool
	}{{
		name:       "nil arg",
		args:       args{},
		wantClient: nil,
		wantErr:    true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClient, err := NewKubernetesClientWithToken(tt.args.token, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKubernetesClientWithToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotClient, tt.wantClient) {
				t.Errorf("NewKubernetesClientWithToken() gotClient = %v, want %v", gotClient, tt.wantClient)
			}
			if gotClient != nil {
				assert.Equal(t, tt.args.master, gotClient.Master())
			}
		})
	}

	client, err := NewKubernetesClientWithToken("token", "master")
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestNewKubernetesClient(t *testing.T) {
	type args struct {
		options *KubernetesOptions
	}
	tests := []struct {
		name       string
		args       args
		wantClient Client
		wantErr    bool
	}{{
		name:       "nil arg",
		args:       args{},
		wantClient: nil,
		wantErr:    false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClient, err := NewKubernetesClient(tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKubernetesClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotClient, tt.wantClient) {
				t.Errorf("NewKubernetesClient() gotClient = %v, want %v", gotClient, tt.wantClient)
			}
		})
	}
}
