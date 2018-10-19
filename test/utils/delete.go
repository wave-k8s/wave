/*
Copyright 2018 Pusher Ltd.

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

package utils

import (
	"context"
	"time"

	g "github.com/onsi/gomega"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteAll lists and deletes all resources
func DeleteAll(cfg *rest.Config, timeout time.Duration, objLists ...runtime.Object) {
	c, err := client.New(rest.CopyConfig(cfg), client.Options{})
	g.Expect(err).ToNot(g.HaveOccurred())
	for _, objList := range objLists {
		g.Eventually(func() error {
			return c.List(context.TODO(), &client.ListOptions{}, objList)
		}, timeout).Should(g.Succeed())
		objs, err := apimeta.ExtractList(objList)
		g.Expect(err).ToNot(g.HaveOccurred())
		errs := make(chan error, len(objs))
		for _, obj := range objs {
			go func(o runtime.Object) {
				errs <- c.Delete(context.TODO(), o)
			}(obj)
		}
		for range objs {
			g.Expect(<-errs).ToNot(g.HaveOccurred())
		}
	}
}
