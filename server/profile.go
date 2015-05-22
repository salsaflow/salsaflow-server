package server

type userProfile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func unmarshalProfile(s sessions.Session) (*userProfile, error) {
	v := s.Get(keyUserProfile)
	if v == nil {
		return nil
	}

	data := v.([]byte)
	var profile userProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func getUserProfile(config *oauth2.Config, token *oauth2.Token) (*userProfile, error) {
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
