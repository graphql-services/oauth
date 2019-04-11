package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-session/session"
	"github.com/google/uuid"
	oauth2gorm "github.com/techknowlogick/go-oauth2-gorm"
	"gopkg.in/oauth2.v3/models"
	"gopkg.in/oauth2.v3/store"

	"gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/server"

	"github.com/graphql-services/oauth/database"
)

// https://auth0.com/docs/quickstart/backend/golang/01-authorization

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		panic(fmt.Errorf(""))
	}

	db := database.NewDBWithString(databaseURL)

	userStore := UserStore{db}
	if err := userStore.Automigrate(); err != nil {
		panic(err)
	}

	manager := manage.NewDefaultManager()
	manager.SetAuthorizeCodeTokenCfg(manage.DefaultAuthorizeCodeTokenCfg)

	// token memory store
	dbStore := oauth2gorm.NewStoreWithDB(&oauth2gorm.Config{}, db, 1800)

	// manager.MustTokenStorage(store.NewMemoryTokenStore())
	manager.MapTokenStorage(dbStore)

	// client memory store
	clientStore := store.NewClientStore()

	clientStore.Set("default", &models.Client{
		Domain: "example.com",
		ID:     "default",
		Secret: "default",
	})

	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)
	manager.SetRefreshTokenCfg(manage.DefaultRefreshTokenCfg)

	rsaKey, err := fetchRSAKey()
	if err != nil {
		panic(err)
	}
	manager.MapAccessGenerate(NewJWTAccessGenerate(rsaKey, jwt.SigningMethodRS256))

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	srv.SetPasswordAuthorizationHandler(func(username, password string) (userID string, err error) {
		ctx := context.Background()
		idpUser, err := FetchIDPUser(ctx, username, password)
		if err != nil {
			return
		}

		user, err := userStore.GetOrCreateUserWithAccount(idpUser.ID, "idp")
		userID = user.ID

		return
	})

	srv.SetUserAuthorizationHandler(userAuthorizeHandler)

	// http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
	// 	err := srv.HandleAuthorizeRequest(w, r)
	// 	if err != nil {
	// 		http.Error(w, err.Error(), http.StatusBadRequest)
	// 	}
	// })
	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleTokenRequest(w, r)
	})

	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
	})

	http.HandleFunc("/credentials", func(w http.ResponseWriter, r *http.Request) {
		clientId := uuid.New().String()[:8]
		clientSecret := uuid.New().String()[:8]
		err := clientStore.Set(clientId, &models.Client{
			ID:     clientId,
			Secret: clientSecret,
			Domain: "http://localhost:9094",
		})
		if err != nil {
			fmt.Println(err.Error())
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"CLIENT_ID": clientId, "CLIENT_SECRET": clientSecret})
	})

	http.HandleFunc("/protected", validateToken(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, I'm protected"))
	}, srv))

	// go testJWKS()
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func validateToken(f http.HandlerFunc, srv *server.Server) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := srv.ValidationBearerToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		f.ServeHTTP(w, r)
	})
}

func userAuthorizeHandler(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	store, err := session.Start(nil, w, r)
	if err != nil {
		return
	}

	uid, ok := store.Get("LoggedInUserID")
	if !ok {
		if r.Form == nil {
			r.ParseForm()
		}

		store.Set("ReturnUri", r.Form)
		store.Save()

		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
		return
	}

	userID = uid.(string)
	store.Delete("LoggedInUserID")
	store.Save()
	return
}
