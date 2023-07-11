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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "kubesphere.io/devops/pkg/api/devops/v1alpha1"
	"kubesphere.io/devops/pkg/api/devops/v1alpha3"
)

// GitRepositoryLister helps list GitRepositories.
type GitRepositoryLister interface {
	// List lists all GitRepositories in the indexer.
	List(selector labels.Selector) (ret []*v1alpha3.GitRepository, err error)
	// GitRepositories returns an object that can list and get GitRepositories.
	GitRepositories(namespace string) GitRepositoryNamespaceLister
	GitRepositoryListerExpansion
}

// gitRepositoryLister implements the GitRepositoryLister interface.
type gitRepositoryLister struct {
	indexer cache.Indexer
}

// NewGitRepositoryLister returns a new GitRepositoryLister.
func NewGitRepositoryLister(indexer cache.Indexer) GitRepositoryLister {
	return &gitRepositoryLister{indexer: indexer}
}

// List lists all GitRepositories in the indexer.
func (s *gitRepositoryLister) List(selector labels.Selector) (ret []*v1alpha3.GitRepository, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha3.GitRepository))
	})
	return ret, err
}

// GitRepositories returns an object that can list and get GitRepositories.
func (s *gitRepositoryLister) GitRepositories(namespace string) GitRepositoryNamespaceLister {
	return gitRepositoryNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// GitRepositoryNamespaceLister helps list and get GitRepositories.
type GitRepositoryNamespaceLister interface {
	// List lists all GitRepositories in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha3.GitRepository, err error)
	// Get retrieves the GitRepository from the indexer for a given namespace and name.
	Get(name string) (*v1alpha3.GitRepository, error)
	GitRepositoryNamespaceListerExpansion
}

// gitRepositoryNamespaceLister implements the GitRepositoryNamespaceLister
// interface.
type gitRepositoryNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all GitRepositories in the indexer for a given namespace.
func (s gitRepositoryNamespaceLister) List(selector labels.Selector) (ret []*v1alpha3.GitRepository, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha3.GitRepository))
	})
	return ret, err
}

// Get retrieves the GitRepository from the indexer for a given namespace and name.
func (s gitRepositoryNamespaceLister) Get(name string) (*v1alpha3.GitRepository, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("gitrepository"), name)
	}
	return obj.(*v1alpha3.GitRepository), nil
}
