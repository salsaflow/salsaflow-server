package memory

import (
	// Internal
	"github.com/salsaflow/salsaflow-server/server/common"

	// Vendor
	"labix.org/v2/mgo/bson"
)

type Store struct {
	byId    map[string]*common.User
	byEmail map[string]*common.User
	byToken map[string]*common.User
}

func NewStore() *Store {
	return &Store{
		byId:    make(map[string]*common.User),
		byEmail: make(map[string]*common.User),
		byToken: make(map[string]*common.User),
	}
}

func (store *Store) FindUserById(id string) (*common.User, error) {
	return store.byId[id], nil
}

func (store *Store) FindUserByEmail(email string) (*common.User, error) {
	return store.byEmail[email], nil
}

func (store *Store) FindUserByToken(token string) (*common.User, error) {
	return store.byToken[token], nil
}

func (store *Store) SaveUser(user *common.User) error {
	if user.Id == "" {
		user.Id = bson.NewObjectId().Hex()
	}

	u := user.Clone()

	store.byId[u.Id] = u
	if u.Email != "" {
		store.byEmail[u.Email] = u
	}
	if u.Token != "" {
		store.byToken[u.Token] = u
	}

	return nil
}

func (store *Store) Close() error {
	return nil
}
