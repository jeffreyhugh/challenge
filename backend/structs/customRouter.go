package structs

import (
	"net/http"

	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

type CustomRouter struct {
	Router *mux.Router
	Docker *client.Client
}

func (k *CustomRouter) HandleFunc(location string, f func(http.ResponseWriter, *http.Request, *CustomRouter)) {
	k.Router.HandleFunc(location, func(u http.ResponseWriter, c *http.Request) {
		f(u, c, k)
	})
}
