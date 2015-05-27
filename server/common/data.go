package common

type User struct {
	Id    string `bson:"_id,omitempty"`
	Name  string `bson:"name,omitempty"`
	Email string `bson:"email,omitempty"`
	Token string `bson:"token,omitempty"`
}

func (user *User) Clone() *User {
	return &User{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
		Token: user.Token,
	}
}
