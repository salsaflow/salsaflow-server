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

func (store *Store) FindUserById(id string) (*common.User, error) {
	return store.find(bson.M{"_id": id})
}

func (store *Store) FindUserByEmail(email string) (*common.User, error) {
	return store.find(bson.M{"email": email})
}

func (store *Store) FindUserByToken(token string) (*common.User, error) {
	return store.find(bson.M{"token": token})
}

func (store *Store) SaveUser(user *common.User) error {
	_, err := store.session.DB("").C("users").UpsertId(user.Id, user)
	return err
}

func (store *Store) Close() error {
	store.session.Close()
	return nil
}

func (store *Store) find(query interface{}) (*common.User, error) {
	var user common.User
	err := store.session.DB("").C("users").Find(query).One(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
