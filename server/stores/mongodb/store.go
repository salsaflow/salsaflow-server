package mongodb

import (
	"github.com/salsaflow/salsaflow-server/server/common"
)

type Store struct {
	s *mgo.Session
}

func NewStore(url string) (*Store, error) {
	s, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Store{s}, nil
}

func (store *Store) SaveUser(user *common.User) error {

}

func (store *Store) FindUserById(id string) (*common.User, error) {

}
