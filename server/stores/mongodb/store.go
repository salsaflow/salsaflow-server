package mongodb

import (
	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type Store struct {
	session *mgo.Session
}

func NewStore(url string) (*Store, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}

	return &Store{session}, nil
}

func (store *Store) FindUserByEmail(email string) (*common.User, error) {
	var user common.User
	err := store.session.DB("").C("users").Find(bson.M{"email": email}).One(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (store *Store) FindUserByToken(email string) (*common.User, error) {
	var user common.User
	err := store.session.DB("").C("users").Find(bson.M{"token": email}).One(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (store *Store) SaveUser(user *common.User) error {
	_, err := store.session.DB("").C("users").UpsertId(user.Id, user)
	return err
}

func (store *Store) Close() error {
	store.session.Close()
	return nil
}
