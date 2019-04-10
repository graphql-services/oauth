// https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
)

type User struct {
	ID        string `gorm:"primary_key"`
	Accounts  []UserAccount
	CreatedAt time.Time
	UpdatedAt time.Time
}
type UserAccount struct {
	ID        string `gorm:"primary_key"`
	Type      string `gorm:"primary_key"`
	UserID    string
	User      User `gorm:"foreignkey:UserID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserStore struct {
	db *gorm.DB
}

func (s *UserStore) Automigrate() error {
	return s.db.AutoMigrate(&User{}, &UserAccount{}).Error
}

func (s *UserStore) GetOrCreateUserWithAccount(accountID, accountType string) (user *User, err error) {
	user, err = s.GetUserByAccount(accountID, accountType)
	if err != nil {
		return
	}
	if user == nil {
		user, err = s.CreateUserWithAccount(accountID, accountType)
	}
	return
}

func (s *UserStore) GetUserByAccount(accountID, accountType string) (user *User, err error) {
	var account UserAccount
	res := s.db.Model(&UserAccount{ID: accountID, Type: accountType}).Preload("User").First(&account)
	if res.RecordNotFound() {
		return
	}
	err = res.Error
	if err != nil {
		return
	}
	user = &account.User
	return
}

func (s *UserStore) CreateUserWithAccount(accountID, accountType string) (user *User, err error) {
	user = &User{
		ID: uuid.New().String(),
		Accounts: []UserAccount{
			UserAccount{
				Type: accountType,
				ID:   accountID,
			},
		},
	}
	err = s.db.Save(user).Error
	return
}
