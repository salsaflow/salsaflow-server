package server

import (
	// Stdlib
	"net/http"

	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	sessions "github.com/goincremental/negroni-sessions"
)

func getRequester(r *http.Request, store DataStore) (*common.User, error) {
	// Token.
	token := r.Header.Get(TokenHeader)
	if token != "" {
		user, err := store.FindUserByToken(token)
		if err != nil {
			return nil, err
		}
		if user != nil {
			return user, nil
		}
	}

	// Session.
	session := sessions.GetSession(r)
	profile, err := unmarshalProfile(session)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		user, err := store.FindUserByEmail(profile.Email)
		if err != nil {
			return nil, err
		}
		if user != nil {
			return user, nil
		}
	}

	// No user found.
	return nil, nil
}
