// +build !heroku

package main

import (
	// Stdlib
	"os"

	// Internal
	"github.com/salsaflow/salsaflow-server/server"
	"github.com/salsaflow/salsaflow-server/server/stores/mongodb"

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
		clientId     = mustGetenv("SF_OAUTH2_CLIENT_ID")
		clientSecret = mustGetenv("SF_OAUTH2_CLIENT_SECRET")
		redirectURL  = mustGetenv("SF_OAUTH2_REDIRECT_URL")

		rootDir  = os.Getenv("SF_ROOT_DIR")
		mongoURL = os.Getenv("SF_MONGODB_URL")
	)

	if mongoURL == "" {
		mongoURL = "localhost:27017"
	}
	store, err := mongodb.NewStore(mongoURL)
	if err != nil {
		return err
	}

	oauth2Config := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"email"},
	}

	return server.New(store, oauth2Config,
		server.SetAddress(addr), server.SetRootDirectory(rootDir)), nil
}
