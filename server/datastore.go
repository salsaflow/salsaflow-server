package server

type DataStore interface {
	GetToken(userId string) (token string, err error)
	SetToken(userId, token string) (err error)
}
