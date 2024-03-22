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

package utils

import (
	"context"
	"time"

	g "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteAll lists and deletes all resources
func DeleteAll(cfg *rest.Config, timeout time.Duration, objLists ...client.ObjectList) {
	c, err := client.New(rest.CopyConfig(cfg), client.Options{})
	g.Expect(err).ToNot(g.HaveOccurred())
	for _, objList := range objLists {
		g.Eventually(func() error {
			return c.List(context.TODO(), objList)
		}, timeout).Should(g.Succeed())

		objs, err := meta.ExtractList(objList)
		g.Expect(err).ToNot(g.HaveOccurred())

		for _, obj := range objs {
			o, ok := obj.(client.Object)
			g.Expect(ok).To(g.BeTrue(), "Expected object to implement client.Object")

			err := c.Delete(context.TODO(), o, &client.DeleteOptions{
				GracePeriodSeconds: metav1.NewDeleteOptions(0).GracePeriodSeconds,
			})
			g.Expect(err).ToNot(g.HaveOccurred())
		}
	}
}
