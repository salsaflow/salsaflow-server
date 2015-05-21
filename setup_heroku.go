// +build heroku

package main

import (
	// Stdlib
	"os"
	"path"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"

	// Vendor
	oauth2 "github.com/goincremental/negroni-oauth2"
)

func NewServer() *server.Server {
	var (
		addr = ":" + os.Getenv("PORT")

		canonicalURL = os.Getenv("CANONICAL_URL")

		clientId     = os.Getenv("OAUTH2_CLIENT_ID")
		clientSecret = os.Getenv("OAUTH2_CLIENT_SECRET")
	)

	oauth2Config := &oauth2.Config{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  path.Join(canonicalURL, "index.html"),
		Scopes:       []string{"https://www.googleapis.com/auth/plus.login"},
	}

	return server.New(oauth2Config, server.SetAddress(addr))
}
