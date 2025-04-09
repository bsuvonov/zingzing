package main


import (
	"net/http"
	"encoding/json"
	"github.com/bsuvonov/zingzing/internal/auth"
	"context"
	"errors"
	"github.com/bsuvonov/zingzing/internal/database"
	"time"
)


func (cfg *apiConfig) putUsersHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}
	user_id, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidJWT) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(401)
			w.Write([]byte("Unauthorized"))
			return
		} else {
			handleError(w, r, err)
			return
		}
	}
	
	hashed_pwd, err := auth.HashPassword(params.Password)
	if err != nil {
		handleError(w, r, err)
		return
	}
	err = cfg.dbq.UpdateUser(context.Background(), database.UpdateUserParams{Email: params.Email, UpdatedAt: time.Now(), HashedPassword: hashed_pwd, ID: user_id})
	if err != nil {
		handleError(w, r, err)
	}

	type returnVals struct {
		Email string `json:"email"`
	}
	responseBody := returnVals {Email: params.Email}
	dat, err := json.Marshal(responseBody)
	if err != nil {
		handleError(w, r, err)
	}
	w.WriteHeader(200)
	w.Write(dat)
}

