package server

import (
	// Stdlib
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	noauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"golang.org/x/oauth2"
)

var (
	errTokenMissing   = errors.New("session token is missing")
	errProfileMissing = errors.New("session profile is missing")
)

// getUserFromToken returns the user record associated with the current API token.
func getUserFromToken(r *http.Request, store DataStore) (*common.User, error) {
	token := r.Header.Get(TokenHeader)
	if token == "" {
		return nil, nil
	}
	return store.FindUserByToken(token)
}

// getUserFromSession returns the user record associated with the current session.
// In case there is no session token, errTokenMissing is returned.
// In case there is no profile stored in the session, errProfileMissing is returned.
func getUserFromSession(r *http.Request, store DataStore) (*common.User, error) {
	var (
		session = sessions.GetSession(r)
		token   = noauth2.GetToken(r)
	)

	// Check the session token first.
	// In case the token is not valid, we clean up the session and return.
	if token == nil || !token.Valid() {
		deleteProfile(session)
		return nil, errTokenMissing
	}

	// Otherwise we try to unmarshal the profile from the session
	// and load the user record from the database.
	profile, err := unmarshalProfile(session)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, errProfileMissing
	}
	return store.FindUserByEmail(profile.Email)
}

// getPageRequester returns the user record for the given request.
// In case the session or the user profile is not complete, getPageRequester fixes stuff.
func getPageRequester(r *http.Request, config *noauth2.Config, store DataStore) (*common.User, error) {
	// Token first.
	user, err := getUserFromToken(r, store)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	// Session.
	user, err = getUserFromSession(r, store)
	switch {
	case err == errTokenMissing:
		// In case the session token is missing, there is nothing we can do.
		// The user is not authenticated and there is no way how to identify them.
		return nil, nil

	case err == errProfileMissing:
		// In case the token is valid, but there is no session profile,
		// we need to fetch that profile from Google.
		var (
			session = sessions.GetSession(r)
			token   = noauth2.GetToken(r)
			cfg     = (*oauth2.Config)(config)
			tok     = (oauth2.Token)(token.Get())
		)
		profile, err := fetchProfile(cfg, &tok)
		if err != nil {
			return nil, err
		}
		// Store the profile in the session immediately.
		if err := marshalProfile(session, profile); err != nil {
			return nil, err
		}

		// Now that we do have the profile, we can try to fetch the associated record.
		// In case there is no record yet, we create a new user for the given email.
		user, err := store.FindUserByEmail(profile.Email)
		if err != nil {
			return nil, err
		}
		if user != nil {
			return user, nil
		}
		return createUser(profile, store)

	case err != nil:
		return nil, err
	default:
		// No need to do anything, simply return the user record.
		// Might be nil, nil, but that is correct as well.
		return user, nil
	}
}

// getApiRequester is similar to getPageRequester, but it is not trying to finalize
// the session or the user record in case something is missing.
func getApiRequester(r *http.Request, store DataStore) (*common.User, error) {
	// Token first.
	user, err := getUserFromToken(r, store)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}

	// Session.
	user, err = getUserFromSession(r, store)
	switch {
	case err == errTokenMissing:
		fallthrough
	case err == errProfileMissing:
		return nil, nil
	default:
		return user, err
	}
}

func createUser(profile *userProfile, store DataStore) (*common.User, error) {
	user := &common.User{
		Name:  profile.Name,
		Email: profile.Email,
	}
	token, err := generateAccessToken()
	if err != nil {
		return nil, err
	}
	user.Token = token
	if err := store.SaveUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func generateAccessToken() (token string, err error) {
	// Generate new token.
	tok := make([]byte, TokenByteLen)
	if _, err := rand.Read(tok); err != nil {
		return "", err
	}

	// Encode the token using hex and return.
	return hex.EncodeToString(tok), nil
}
