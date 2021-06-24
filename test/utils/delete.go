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
			return c.List(context.TODO(), objList)
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
