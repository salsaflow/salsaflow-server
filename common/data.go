package common

type User struct {
	Name  string `bson:"name,omitempty"`
	Email string `bson:"email,omitempty"`
	Token string `bson:"token,omitempty"`
}
