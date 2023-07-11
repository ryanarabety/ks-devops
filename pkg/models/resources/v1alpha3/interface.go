/*
Copyright 2020 KubeSphere Authors

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
	"reflect"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubesphere.io/devops/pkg/api"
	"kubesphere.io/devops/pkg/apiserver/query"
)

type Interface interface {
	// Get retrieves a single object by its namespace and name
	Get(namespace, name string) (runtime.Object, error)

	// List retrieves a collection of objects matches given query
	List(namespace string, query *query.Query) (*api.ListResult, error)
}

// CompareFunc return true is left great than right
type CompareFunc func(runtime.Object, runtime.Object, query.Field) bool

type FilterFunc func(runtime.Object, query.Filter) bool

var alwaysTrueFilter FilterFunc = func(object runtime.Object, filter query.Filter) bool {
	return true
}

var alwaysFalseFilter FilterFunc = func(object runtime.Object, filter query.Filter) bool {
	return false
}

// And performs like logical operation a && b
func (ff FilterFunc) And(anotherFf FilterFunc) FilterFunc {
	return func(object runtime.Object, filter query.Filter) bool {
		if ff == nil {
			ff = alwaysTrueFilter
		}
		if anotherFf == nil {
			anotherFf = alwaysTrueFilter
		}
		return ff(object, filter) && anotherFf(object, filter)
	}
}

// Or performs like logical operation a || b
func (ff FilterFunc) Or(anotherFf FilterFunc) FilterFunc {
	return func(object runtime.Object, filter query.Filter) bool {
		if ff == nil {
			ff = alwaysFalseFilter
		}
		if anotherFf == nil {
			anotherFf = alwaysFalseFilter
		}
		return ff(object, filter) || anotherFf(object, filter)
	}
}

type TransformFunc func(runtime.Object) interface{}

// NoTransformFunc keeps original data without any transformation.
func NoTransformFunc() TransformFunc {
	return func(object runtime.Object) interface{} {
		return object
	}
}

// ToListResult converts array of runtime.Object to ListResult, if the handler is not provided, we will
// create a defaultListHandler to handle the list.
func ToListResult(objects []runtime.Object, q *query.Query, handler ListHandler) *api.ListResult {
	if handler == nil {
		handler = defaultListHandler{}
	}
	return DefaultList(objects, q, handler.Comparator(), handler.Filter(), handler.Transformer())
}

func DefaultList(objects []runtime.Object, q *query.Query, compareFunc CompareFunc, filterFunc FilterFunc, transformFuncs ...TransformFunc) *api.ListResult {
	// selected matched ones
	var filtered []runtime.Object
	// filter objects
	for _, object := range objects {
		if object == nil || reflect.ValueOf(object).IsNil() {
			continue
		}
		selected := true
		for field, value := range q.Filters {
			if filterFunc != nil && !filterFunc(object, query.Filter{Field: field, Value: value}) {
				selected = false
				break
			}
		}
		if selected {
			filtered = append(filtered, object)
		}
	}

	// sort by sortBy field
	if compareFunc != nil {
		sort.Slice(filtered, func(i, j int) bool {
			if !q.Ascending {
				return compareFunc(filtered[i], filtered[j], q.SortBy)
			}
			return !compareFunc(filtered[i], filtered[j], q.SortBy)
		})
	}

	total := len(filtered)

	if q.Pagination == nil {
		q.Pagination = query.NoPagination
	}

	start, end := q.Pagination.GetValidPagination(total)

	// transform objects
	var result = make([]interface{}, end-start)
	transformFuncs = nilFilter(transformFuncs)
	if len(transformFuncs) == 0 {
		transformFuncs = append(transformFuncs, NoTransformFunc())
	}
	for i, obj := range filtered[start:end] {
		var transferred interface{}
		for _, transform := range transformFuncs {
			transferred = transform(obj)
			if transferredObj, ok := transferred.(runtime.Object); ok {
				obj = transferredObj
			}
		}
		result[i] = transferred
	}

	return api.NewListResult(result, total)
}

// DefaultCompare creates a default ObjectMeta compare function.
func DefaultCompare() CompareFunc {
	return func(left runtime.Object, right runtime.Object, field query.Field) bool {
		leftOma, ok := left.(metav1.ObjectMetaAccessor)
		if !ok {
			return false
		}
		rightOma, ok := right.(metav1.ObjectMetaAccessor)
		if !ok {
			return false
		}
		return DefaultObjectMetaCompare(leftOma.GetObjectMeta(), rightOma.GetObjectMeta(), field)
	}
}

// NameCompare returns a compare function that compare by name
func NameCompare() CompareFunc {
	return func(left runtime.Object, right runtime.Object, field query.Field) bool {
		leftOma, ok := left.(metav1.ObjectMetaAccessor)
		if !ok {
			return false
		}
		rightOma, ok := right.(metav1.ObjectMetaAccessor)
		if !ok {
			return false
		}
		field = "!" + query.FieldName
		return DefaultObjectMetaCompare(leftOma.GetObjectMeta(), rightOma.GetObjectMeta(), field)
	}
}

// DefaultFilter creates a default ObjectMeta filter function.
func DefaultFilter() FilterFunc {
	return func(obj runtime.Object, filter query.Filter) bool {
		oma, ok := obj.(metav1.ObjectMetaAccessor)
		if !ok || oma == nil || reflect.ValueOf(oma).IsNil() {
			return false
		}
		return DefaultObjectMetaFilter(oma.GetObjectMeta(), filter)
	}
}

// DefaultObjectMetaCompare return true is left greater than right
func DefaultObjectMetaCompare(left, right metav1.Object, sortBy query.Field) bool {
	switch sortBy {
	// ?sortBy=name
	case query.FieldName:
		// sort the name in descending order
		return strings.Compare(left.GetName(), right.GetName()) > 0
	case "!" + query.FieldName:
		// sort the name in ascending order
		return strings.Compare(left.GetName(), right.GetName()) < 0
	//	?sortBy=creationTimestamp
	default:
		fallthrough
	case query.FieldCreationTimeStamp:
		// compare by name if creation timestamp is equal
		leftTime := left.GetCreationTimestamp()
		rightTime := right.GetCreationTimestamp()
		if leftTime.Equal(&rightTime) {
			return strings.Compare(left.GetName(), right.GetName()) > 0
		}
		return leftTime.After(rightTime.Time)
	}
}

// DefaultObjectMetaFilter filters data with given filter
func DefaultObjectMetaFilter(item metav1.Object, filter query.Filter) bool {
	switch filter.Field {
	case query.FieldNames:
		for _, name := range strings.Split(string(filter.Value), ",") {
			if item.GetName() == name {
				return true
			}
		}
		return false
	// /namespaces?page=1&limit=10&name=default
	case query.FieldName:
		return strings.Contains(item.GetName(), string(filter.Value))
		// /namespaces?page=1&limit=10&uid=a8a8d6cf-f6a5-4fea-9c1b-e57610115706
	case query.FieldUID:
		return string(item.GetUID()) == string(filter.Value)
		// /deployments?page=1&limit=10&namespace=kubesphere-system
	case query.FieldNamespace:
		return item.GetNamespace() == string(filter.Value)
		// /namespaces?page=1&limit=10&ownerReference=a8a8d6cf-f6a5-4fea-9c1b-e57610115706
	case query.FieldOwnerReference:
		for _, ownerReference := range item.GetOwnerReferences() {
			if string(ownerReference.UID) == string(filter.Value) {
				return true
			}
		}
		return false
		// /namespaces?page=1&limit=10&ownerKind=Workspace
	case query.FieldOwnerKind:
		for _, ownerReference := range item.GetOwnerReferences() {
			if ownerReference.Kind == string(filter.Value) {
				return true
			}
		}
		return false
		// /namespaces?page=1&limit=10&annotation=openpitrix_runtime
	case query.FieldAnnotation:
		return labelsMatch(item.GetAnnotations(), string(filter.Value))
		// /namespaces?page=1&limit=10&label=kubesphere.io/workspace=system-workspace
	case query.FieldLabel:
		return labelsMatch(item.GetLabels(), string(filter.Value))
	default:
		// We should allow fields that are not found
		return true
	}
}

// labelsMatch handles multi-label value pairs split by ",".
// e.g. devops.ks.io/creator=admin,devops.ks.io/status=success
func labelsMatch(labels map[string]string, filterStr string) bool {
	filters := strings.Split(filterStr, ",")
	var match = true
	for _, filter := range filters {
		match = match && labelMatch(labels, strings.TrimSpace(filter))
		if !match {
			break
		}
	}
	return match
}

func labelMatch(labels map[string]string, filter string) bool {
	fields := strings.SplitN(filter, "=", 2)
	var key, value string
	var opposite bool
	if len(fields) == 2 {
		key = fields[0]
		if strings.HasSuffix(key, "!") {
			key = strings.TrimSuffix(key, "!")
			opposite = true
		}
		value = fields[1]
	} else {
		key = fields[0]
		value = "*"
	}
	for k, v := range labels {
		if opposite {
			if (k == key) && v != value {
				return true
			}
		} else {
			if (k == key) && (value == "*" || v == value) {
				return true
			}
		}
	}
	return false
}

func nilFilter(transformFuncs []TransformFunc) []TransformFunc {
	var result []TransformFunc
	for i := range transformFuncs {
		if transformFuncs[i] != nil {
			result = append(result, transformFuncs[i])
		}
	}
	return result
}
