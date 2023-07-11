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
package kapis

import (
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"io"
	"kubesphere.io/devops/pkg/server/errors"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestIgnoreEOF(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want error
	}{{
		name: "Should return nil if error is io.EOF",
		args: args{
			err: io.EOF,
		},
		want: nil,
	}, {
		name: "Should return the same error if error is not io.EOF",
		args: args{
			errors.New("Fake Error"),
		},
		want: errors.New("Fake Error"),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := IgnoreEOF(tt.args.err); !reflect.DeepEqual(err, tt.want) {
				t.Errorf("IgnoreEOF() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestResponseWriter_WriteEntityOrError(t *testing.T) {
	type fakeType struct {
		Name string `json:"name"`
	}
	type args struct {
		entity interface{}
		err    error
	}
	tests := []struct {
		name         string
		args         args
		wantErr      bool
		wantResponse string
	}{{
		name: "Should response correctly when no error",
		args: args{
			entity: fakeType{Name: "fake-name"},
			err:    nil,
		},
		wantResponse: `{"name":"fake-name"}`,
	}, {
		name: "Should response error when err is not nil",
		args: args{
			entity: fakeType{Name: "fake-name"},
			err:    errors.New("fake error occurred"),
		},
		wantErr:      true,
		wantResponse: "fake error occurred\n",
	},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			response := restful.NewResponse(recorder)
			response.SetRequestAccepts(restful.MIME_JSON)
			re := ResponseWriter{
				response,
			}
			re.WriteEntityOrError(tt.args.entity, tt.args.err)
			if tt.wantErr {
				assert.False(t, recorder.Code >= 200 && recorder.Code < 300)
				assert.Equal(t, tt.wantResponse, recorder.Body.String())
				return
			}
			got := recorder.Body.String()
			assert.JSONEq(t, got, tt.wantResponse)
		})
	}
}
