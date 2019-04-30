// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"context"
	"time"

	"github.com/graphql-services/oauth/database"
)

type UserAccount struct {
	ID        string     `gorm:"primary_key"`
	Type      string     `json:"type" gorm:"primary_key"`
	UpdatedAt *time.Time `json:"updatedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UserID    string     `json:"user_id"`
}

type UserStore struct {
	DB *database.DB
	ID *IDClient
}

func (s *UserStore) AutoMigrate() error {
	return s.DB.AutoMigrate(&UserAccount{})
}

func (s *UserStore) GetOrCreateUserWithAccount(accountID, email, accountType string) (user *User, err error) {
	user, err = s.GetUserByAccount(accountID, accountType)
	if err != nil {
		return
	}

	if user == nil {
		user, err = s.CreateUserWithAccount(accountID, email, accountType)
		if err != nil {
			return
		}
	}
	return
}

func (s *UserStore) GetUser(userID string) (user *User, err error) {
	user, err = s.ID.GetUser(context.Background(), userID)
	return
}

func (s *UserStore) GetUserByAccount(accountID, accountType string) (user *User, err error) {
	var account UserAccount
	res := s.DB.Client().Model(&UserAccount{}).First(&account, &UserAccount{ID: accountID, Type: accountType})
	if res.RecordNotFound() {
		return
	}
	err = res.Error
	if err != nil {
		return
	}
	user, err = s.ID.GetUser(context.Background(), account.UserID)
	return
}

func (s *UserStore) CreateUserWithAccount(accountID, email, accountType string) (user *User, err error) {
	user, err = s.ID.InviteUser(context.Background(), email)
	if err != nil {
		return
	}

	account := UserAccount{Type: accountType, ID: accountID, UserID: user.ID}
	err = s.DB.Client().FirstOrCreate(&account, account).Error

	return
}
