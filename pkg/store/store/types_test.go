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

package store

import "testing"

func TestStepLogKey(t *testing.T) {
	type args struct {
		stage int
		step  int
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "normal",
		args: args{
			stage: 1,
			step:  2,
		},
		want: "log-step-1-2",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StepLogKey(tt.args.stage, tt.args.step); got != tt.want {
				t.Errorf("StepLogKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
