package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-session/session"
	"github.com/google/uuid"

	opentracing "github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/rs/cors"
	oauth2gorm "github.com/techknowlogick/go-oauth2-gorm"
	"gopkg.in/oauth2.v3"
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
		panic(fmt.Errorf("Missing DATABASE_URL environment variable"))
	}

	db := database.NewDBWithString(databaseURL)

	idp := NewIDPClient()
	id := NewIDClient()
	userStore := UserStore{DB: db, ID: id}
	if err := userStore.AutoMigrate(); err != nil {
		panic(err)
	}

	t := Tracer{}
	t.Initialize()
	defer t.Close()

	manager := manage.NewDefaultManager()
	manager.SetAuthorizeCodeTokenCfg(manage.DefaultAuthorizeCodeTokenCfg)

	// token memory store
	dbStore := oauth2gorm.NewStoreWithDB(&oauth2gorm.Config{}, db.Client(), 1800)

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

	manager.SetPasswordTokenCfg(&manage.Config{AccessTokenExp: time.Second * time.Duration(getEnvInt("ACCESS_TOKEN_EXPIRE_IN", 7200)), RefreshTokenExp: time.Hour * 24 * 7, IsGenerateRefresh: true})
	manager.SetRefreshTokenCfg(manage.DefaultRefreshTokenCfg)

	manager.MapAccessGenerate(NewJWTAccessGenerate(jwt.SigningMethodRS256, &userStore))

	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
		re = &errors.Response{
			Error: err,
		}
		log.Println("Internal Error:", err.Error())
		return
	})

	srv.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	srv.SetPasswordAuthorizationHandler(func(username, password string) (userID string, err error) {
		ctx := context.Background()

		span, ctx := opentracing.StartSpanFromContext(ctx, "oauth - /authorization/password")
		defer span.Finish()
		span.LogFields(
			otlog.String("username", username))

		idpUser, err := idp.FetchIDPUserFromOIDC(ctx, username, password)
		if err != nil {
			return
		}
		fmt.Println("??", idpUser, err)

		if idpUser == nil {
			idpUser, err = idp.FetchIDPUser(ctx, username, password)
			if err != nil {
				return
			}
		}
		if idpUser != nil {
			user, _err := userStore.GetOrCreateUserWithAccount(ctx, idpUser.ID, username, "idp")
			if _err != nil {
				err = _err
				return
			}
			if user != nil {
				userID = user.ID
			}
		}

		return
	})

	srv.SetUserAuthorizationHandler(userAuthorizeHandler)
	srv.SetClientInfoHandler(func(r *http.Request) (clientID, clientSecret string, err error) {
		clientID, clientSecret, _ = r.BasicAuth()
		return
	})
	srv.ExtensionFieldsHandler = func(ti oauth2.TokenInfo) (fieldsValue map[string]interface{}) {
		scope := ti.GetScope()
		if containsScope(scope, "openid") {
			fieldsValue = map[string]interface{}{}
			idToken, err := generateIDToken(context.Background(), ti, &userStore)
			if err != nil {
				panic(err)
			}
			fieldsValue["id_token"] = idToken
		}
		return
	}

	mux := http.NewServeMux()

	// http://localhost:8080/authorize?client_id=default&redirect_uri=https%3A%2F%2Fwww.example.com&response_type=code&state=somestate&scope=read_write
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		err := srv.HandleAuthorizeRequest(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		srv.HandleTokenRequest(w, r)
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is login form"))
	})

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
	})

	mux.HandleFunc("/credentials", func(w http.ResponseWriter, r *http.Request) {
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

	// mux.HandleFunc("/protected", validateToken(func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("Hello, I'm protected"))
	// }, srv))

	handler := cors.AllowAll().Handler(mux)

	// go testJWKS()
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	log.Printf("connect to http://localhost:%s/", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// func validateToken(f http.HandlerFunc, srv *server.Server) http.HandlerFunc {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		_, err := srv.ValidationBearerToken(r)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}

// 		f.ServeHTTP(w, r)
// 	})
// }

func userAuthorizeHandler(w http.ResponseWriter, r *http.Request) (userID string, err error) {
	store, err := session.Start(nil, w, r)
	if err != nil {
		return
	}

	uid, ok := store.Get("LoggedInUserID") // OR get value from url querystring
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

func getEnvInt(name string, defaultValue int) int {
	val := os.Getenv(name)
	v, err := strconv.Atoi(val)
	if err != nil {
		v = defaultValue
	}
	return v
}
