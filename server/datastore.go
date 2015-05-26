package server

type DataStore interface {
	FindUserByEmail(email string) (*common.User, error)
	FindUserByToken(token string) (*common.User, error)
	SaveUser(user *common.User) error
}
