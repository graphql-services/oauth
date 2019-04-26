// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"testing"

	"github.com/graphql-services/oauth/database"
)

func TestUserStore(t *testing.T) {
	db := database.NewDBWithString("sqlite3://:memory:")

	s := UserStore{db}

	if err := s.Automigrate(); err != nil {
		t.Errorf("[%v] Failed to automigrate", err.Error())
	}

	accountType := "facebook"
	accountID := "abcd1234"
	email := "john.doe@example.com"

	u, err := s.GetUserByAccount(accountID, accountType)
	if err != nil {
		t.Errorf("[%v] Failed to get user by account", err.Error())
	}
	if u != nil {
		t.Errorf("[%v] user should be nil, but found", u.ID)
	}

	u, err = s.CreateUserWithAccount(accountID, email, accountType)
	if err != nil {
		t.Errorf("[%v] Failed to create user", err.Error())
	}
	if u == nil {
		t.Errorf("[%v] user should not be nil", u.ID)
	}

	u2, err := s.GetOrCreateUserWithAccount(accountID, email, accountType)
	if err != nil {
		t.Errorf("[%v] Failed to get or create user", err.Error())
	}

	if u.ID != u2.ID {
		t.Errorf("[%v == %v] fetched users IDs should be equal", u.ID, u2.ID)
	}
}
