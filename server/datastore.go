package server

import (
	"github.com/salsaflow/salsaflow-server/server/common"
)

type DataStore interface {
	FindUserById(id string) (*common.User, error)
	FindUserByEmail(email string) (*common.User, error)
	FindUserByToken(token string) (*common.User, error)
	SaveUser(user *common.User) error
	Close() error
}
