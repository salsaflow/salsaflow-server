// +build !heroku

package main

import (
	// Stdlib
	"flag"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"
)

func NewServer() *server.Server {
	flagAddress := flag.String("addr", server.DefaultAddress, "network address to listen on")
	flagTimeout := flag.Duration("timeout", server.DefaultTimeout, "server shutdown timeout")
	flag.Parse()

	return server.New(
		server.SetAddress(*flagAddress),
		server.SetShutdownTimeout(*flagTimeout))
}
