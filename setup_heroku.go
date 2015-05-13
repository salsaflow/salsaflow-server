// +build heroku

package main

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"
)

func NewServer() *server.Server {
	var (
		addr = ":" + os.Getenv("PORT")
	)

	return server.New(server.SetAddress(addr))
}
