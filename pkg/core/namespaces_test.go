/*
Copyright 2018 Pusher Ltd. and Wave Contributors

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

package core

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

var _ = Describe("Wave Namespace Suite", func() {
	Context("BuildCacheDefaultNamespaces", func() {
		It("Returns an empty config for an empty string", func() {
			Expect(BuildCacheDefaultNamespaces("")).To(Equal(map[string]cache.Config(nil)))
		})

		It("Returns a single entry for one namespace", func() {
			Expect(BuildCacheDefaultNamespaces("test")).To(Equal(map[string]cache.Config{
				"test": {},
			}))
		})

		It("Returns a multiple entries for a list of namespaces", func() {
			Expect(BuildCacheDefaultNamespaces("test,ns1,ns2")).To(Equal(map[string]cache.Config{
				"test": {},
				"ns1":  {},
				"ns2":  {},
			}))
		})
	})
})
