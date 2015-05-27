package server

import (
	// Stdlib
	"crypto/rand"
	"encoding/hex"
	"net/http"

	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	noauth2 "github.com/goincremental/negroni-oauth2"
	sessions "github.com/goincremental/negroni-sessions"
	"golang.org/x/oauth2"
)

func getRequester(r *http.Request, config *noauth2.Config, store DataStore) (*common.User, error) {
	// Token first.
	// In case there is a valid access token present in the request, we are done.
	// That means that the user is fully initialised and there is nothing to be done.
	tokenHeader := r.Header.Get(TokenHeader)
	if tokenHeader != "" {
		user, err := store.FindUserByToken(tokenHeader)
		if err != nil {
			return nil, err
		}
		if user != nil {
			return user, nil
		}
	}

	// Session.
	// In case there is a session, the user still does not have to be initialised.
	// So we need to make sure there is a user record in the DB. In case there isn't,
	// we create one.
	var (
		session = sessions.GetSession(r)
		token   = noauth2.GetToken(r)
	)

	// Check the session token first.
	// In case the token is not valid, we clean up the session and return.
	if token == nil || !token.Valid() {
		deleteProfile(session)
		return nil, nil
	}

	// Otherwise we try to unmarshal the profile from the session
	// and load the user record from the database.
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

	// Now we know that the token is valid,
	// but there is no profile stored in the session, so let's fetch it
	// and store it in the session.
	var (
		cfg = (*oauth2.Config)(config)
		tok = (oauth2.Token)(token.Get())
	)
	profile, err = fetchProfile(cfg, &tok)
	if err != nil {
		return nil, err
	}
	if err := marshalProfile(session, profile); err != nil {
		return nil, err
	}

	// Now that we do have a user profile, we can try to fetch the record.
	// In case there is no record yet, we create a new user for the given email.
	user, err := store.FindUserByEmail(profile.Email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return user, nil
	}
	return createUser(profile, store)
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
