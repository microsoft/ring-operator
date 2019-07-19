package controller

import (
	"ring-operator/pkg/controller/ring"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, ring.Add)
}
