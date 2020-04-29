package controller

import (
	"github.com/Dysproz/guestbook-operator/pkg/controller/guestbook"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, guestbook.Add)
}
