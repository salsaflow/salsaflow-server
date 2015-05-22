package server

import (
	// Stdlib
	"encoding/json"

	// Vendor
	sessions "github.com/goincremental/negroni-sessions"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/plus/v1"
)

const keyUserProfile = "userProfile"

type userProfile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func unmarshalProfile(s sessions.Session) (*userProfile, error) {
	// Get the profile value from the session.
	v := s.Get(keyUserProfile)
	if v == nil {
		return nil, nil
	}

	// Unmarshall the profile.
	data := v.([]byte)
	var profile userProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func marshalProfile(s sessions.Session, profile *userProfile) error {
	// Marshal the profile.
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	// Save it into the session.
	s.Set(keyUserProfile, data)
	return nil
}

func fetchProfile(config *oauth2.Config, token *oauth2.Token) (*userProfile, error) {
	// Get the HTTP client to use for the request.
	httpClient := config.Client(context.Background(), token)

	// Instantiate a service.
	service, err := plus.New(httpClient)
	if err != nil {
		return nil, err
	}

	// Instantiate the People service.
	people := plus.NewPeopleService(service)
	me, err := people.Get("me").Do()
	if err != nil {
		return nil, err
	}

	// Return the requested data.
	var (
		name  = me.DisplayName
		email string
	)
	for _, e := range me.Emails {
		if e.Type == "account" {
			email = e.Value
		}
	}
	return &userProfile{
		Name:  name,
		Email: email,
	}, nil
}
