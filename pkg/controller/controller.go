package controller

import (
	"github.com/microsoft/ring-operator/pkg/controller/ingressroute"
	"github.com/microsoft/ring-operator/pkg/controller/ring"
	"github.com/microsoft/ring-operator/pkg/controller/service"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	AddToManagerFuncs = append(AddToManagerFuncs, ring.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, service.Add)
	AddToManagerFuncs = append(AddToManagerFuncs, ingressroute.Add)
}