package controller

import (
	"github.com/wave-k8s/wave/pkg/controller/daemonset"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, daemonset.Add)
}
