package database

import (
	"net/url"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// NewDBWithString ...
func NewDBWithString(urlString string) *gorm.DB {
	u, err := url.Parse(urlString)
	if err != nil {
		panic(err)
	}

	urlString = strings.Replace(urlString, u.Scheme+"://", "", 1)

	db, err := gorm.Open(u.Scheme, urlString)
	if err != nil {
		panic(err)
	}
	// db.LogMode(true)
	return db
}
