package main


import (
	"net/http"
	"encoding/json"
	"github.com/google/uuid"
	"context"
	"github.com/bsuvonov/zingzing/internal/database"
	"errors"
	"time"
	"github.com/bsuvonov/zingzing/internal/auth"
	"log"
	"fmt"
	"strconv"
	"database/sql"
)


func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.And(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}


func (cfg *apiConfig) postUsersHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
		return
	}

	if len(params.Password) < 1 { 
		handleError(w, r, errors.New("password must be longer than 6 characters"))
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		handleError(w, r, err)
		return
	}

	type returnVals struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		IsPremium bool	`json:"is_premium"`
	}
	respBody := returnVals{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email: params.Email,
		IsPremium: false,
	}

	_, err = cfg.dbq.CreateUser(context.Background(), database.CreateUserParams{ID: respBody.ID, CreatedAt: respBody.CreatedAt, UpdatedAt: respBody.UpdatedAt, Email: respBody.Email, HashedPassword: hashedPassword})
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")

	dat, err := json.Marshal(respBody)
	if err!=nil {
		handleError(w, r, err)
		return
	}
	w.Write(dat)
}




func (cfg *apiConfig) zingersPostHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// if len(params.Body) > 140 {
	// 	handleError(w, r, err)
	// 	return
	// }

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		handleError(w, r, err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	// fmt.Println("BODY:" + params.Body)
	// // fmt.Println("ERROR:" + err.Error())
	// fmt.Println("TOKEN:" + token)
	// fmt.Print("HEADER:")
	// fmt.Println(r.Header)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidJWT) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(401)
			w.Write([]byte("Unauthorized"))
		} else {
			handleError(w, r, err)
		}
		return
	}

	params.Body = censorZinger(params.Body)

	type returnVals struct {
		Id uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	respBody := returnVals{
		Id: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body: params.Body,
		UserId: userID,
	}

	cfg.dbq.CreateZinger(context.Background(), database.CreateZingerParams{ID: respBody.Id, CreatedAt: respBody.CreatedAt, UpdatedAt: respBody.UpdatedAt, Body: respBody.Body, UserID: respBody.UserId})

	dat, err := json.Marshal(respBody)
	if err!=nil {
		log.Printf("Error in converting response body to json: %s", err)
		return
	}

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}




func (cfg *apiConfig) userLoginHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
		return
	}

	user, err := cfg.dbq.GetUserByEmail(context.Background(), params.Email)
	if err != nil && err.Error() == "sql: no rows in result set" {
		handleUserLoginError(w, r)
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, params.Password)
	if err != nil {
		handleUserLoginError(w, r)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.jwt_secret, time.Duration(3600)*time.Second)
	if err != nil {
		handleError(w, r, err)
		return
	}

	type returnVals struct {
		Id uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
		Token string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsPremium bool `json:"is_premium"`
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		handleError(w, r, err)
		return
	}
	_, err = cfg.dbq.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{Token: refreshToken, CreatedAt: time.Now(), UpdatedAt: time.Now(), UserID: user.ID, ExpiresAt: time.Now().Add(time.Duration(1440)*time.Hour)})
	if err!= nil {
		handleError(w, r, err)
		return
	}

	respBody := returnVals{Id: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email, Token: token, RefreshToken: refreshToken, IsPremium: user.IsPremium}
	dat, err := json.Marshal(respBody)
	if err != nil {
		handleError(w, r, err)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
}


func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(1000 * time.Millisecond)
	// expects refresh token in request header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		handleError(w, r, err)
		return
	}
	token, err := cfg.dbq.GetRefreshTokenByToken(context.Background(), refreshToken)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(401)
			w.Write([]byte("Unauthorized"))
			return
		}
		handleError(w, r, err)
	}
	fmt.Println("TOKEN:")
	fmt.Print("CREATED AT:")
	fmt.Println(token.CreatedAt.UTC())
	fmt.Printf("UPDATED AT:")
	fmt.Println(token.UpdatedAt.UTC())
	fmt.Print("EXPIRES AT:")
	fmt.Println(token.ExpiresAt.UTC())
	fmt.Print("TOKEN:")
	fmt.Println(token.Token)
	if token.RevokedAt.Valid {
		fmt.Print("REVOKED AT:")
		fmt.Println(token.RevokedAt.Time.UTC())
	}
	fmt.Println("TIME NOW:" + time.Now().UTC().String())
	fmt.Println("ARE WE RETURNING 401:" + strconv.FormatBool(token.RevokedAt.Time.UTC().Before(time.Now().UTC())))
	if (token.RevokedAt.Valid && token.RevokedAt.Time.UTC().Before(time.Now().UTC())) || token.ExpiresAt.UTC().Before(time.Now().UTC()) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
		return
	}

	user, err := cfg.dbq.GetUserByRefreshToken(context.Background(), refreshToken)
	if err!=nil {
		handleError(w, r, err)
		return
	}

	tokenJWT, err := auth.MakeJWT(user.ID, cfg.jwt_secret, time.Duration(1)*time.Hour)
	if err != nil {
		handleError(w, r, err)
		return
	}

	type returnVals struct {
		Token string `json:"token"`
	}

	respBody := returnVals{Token: tokenJWT}

	dat, err := json.Marshal(respBody)
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}


func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err!=nil {
		handleError(w, r, err)
	}
	fmt.Println("============")
	fmt.Println("REVOKING:" + token)
	err = cfg.dbq.RevokeRefreshToken(context.Background(), database.RevokeRefreshTokenParams{UpdatedAt: time.Now().UTC(), RevokedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, Token: token})
	if err != nil {
		handleError(w, r, err)
	}
	tokenDb, _ := cfg.dbq.GetRefreshTokenByToken(context.Background(), token)
	fmt.Println("TOKEN:")
	fmt.Print("CREATED AT:")
	fmt.Println(tokenDb.CreatedAt)
	fmt.Printf("UPDATED AT:")
	fmt.Println(tokenDb.UpdatedAt)
	fmt.Print("EXPIRES AT:")
	fmt.Println(tokenDb.ExpiresAt)
	fmt.Print("TOKEN:")
	fmt.Println(tokenDb.Token)
	if tokenDb.RevokedAt.Valid {
		fmt.Print("REVOKED AT:")
		fmt.Println(tokenDb.RevokedAt.Time)
	}
	fmt.Println("============")
	w.WriteHeader(204)
}



func (cfg *apiConfig) webhookHandler(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		handleErrorUnauthorized(w)
		return
	}
	if cfg.zingpay_key != apiKey {
		handleErrorUnauthorized(w)
		return
	}
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
	}
	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}
	err = cfg.dbq.UpgradeToPremium(context.Background(), uuid.MustParse(params.Data.UserID))
	if err != nil {
		handleErrorNotFound(w)
		return
	}
	w.WriteHeader(204)
}

