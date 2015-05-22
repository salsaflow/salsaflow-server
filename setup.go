// +build !heroku

package main

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"

	// Vendor
	oauth2 "github.com/goincremental/negroni-oauth2"
)

func LoadServerFromEnvironment() (srv *server.Server, err error) {
	mustGetenv := func(key string) (value string) {
		value = os.Getenv(key)
		if value == "" {
			panic(ErrVariableNotSet{key})
		}
		return
	}

	defer func() {
		if r := recover(); r != nil {
			if ex, ok := r.(*ErrVariableNotSet); ok {
				err = ex
			} else {
				panic(r)
			}
		}
	}()

	var (
		addr         = mustGetenv("SF_LISTEN_ADDRESS")
		rootDir      = os.Getenv("SF_ROOT_DIR")
		clientId     = mustGetenv("SF_OAUTH2_CLIENT_ID")
		clientSecret = mustGetenv("SF_OAUTH2_CLIENT_SECRET")
		redirectURL  = mustGetenv("SF_OAUTH2_REDIRECT_URL")
	)

	oauth2Config := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"email"},
	}

	return server.New(oauth2Config, server.SetAddress(addr), server.SetRootDirectory(rootDir)), nil
}
