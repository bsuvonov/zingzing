package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/bsuvonov/zingzing/internal/auth"
	"github.com/bsuvonov/zingzing/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)





func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}




type apiConfig struct {
	fileserverHits atomic.Int32
	dbq *database.Queries
	jwt_secret string
	zingpay_key string
}


func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	val := int(cfg.fileserverHits.Load())

	content := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, ZingZing Admin</h1>
    <p>ZingZing has been visited %d times!</p>
  </body>
</html>`, val)

	w.Write([]byte(content))

}


func (cfg *apiConfig) middlewareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.And(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}


func censorZinger(input string) string {
	// Regex for "stupid" (case-insensitive, optional ! at the end)
	stupid := regexp.MustCompile(`(?i)\bstupid\b!?`)

	// Regex for "kerfuffle" and "fornax" (case-insensitive, word boundaries only)
	dumb := regexp.MustCompile(`(?i)\bdumb\b`)
	idiot := regexp.MustCompile(`(?i)\bidiot\b`)

	output := stupid.ReplaceAllStringFunc(input, func(match string) string {
		return "****"
	})

	output = idiot.ReplaceAllStringFunc(output, func(match string) string {
		return "****"
	})

	output = dumb.ReplaceAllStringFunc(output, func(match string) string {
		return "****"
	})

	return output
	}


func handleError(w http.ResponseWriter, r *http.Request, err error) {

	type returnVals struct {
		Err string `json:"error"`
	}

	respBody := returnVals{
		Err: "Something went wrong",
	}

	if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
		respBody.Err = "User already exists"
	}

	dat, err := json.Marshal(respBody)
	if err!=nil {
		log.Printf("Error in converting response body to json when handling error: %s", err)
		return
	}
	w.WriteHeader(500)
	w.Header().Set("Content-Type", "application/json")
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


func (cfg *apiConfig) zingersGetHandler(w http.ResponseWriter, r *http.Request) {
	authorIDParam := r.URL.Query().Get("author_id")

	var zingers []database.Zinger
	var err error

	authorID, parseErr := uuid.Parse(authorIDParam)
	if parseErr != nil {
		zingers, err = cfg.dbq.GetAllZingers(context.Background())
	} else {
		zingers, err = cfg.dbq.GetZingersByUser(context.Background(), authorID)
	}
	if err != nil {
		handleError(w, r, err)
		return
	}

	sortOrder := r.URL.Query().Get("sort")
	if sortOrder == "desc" {
		sort.Slice(zingers, func(i, j int) bool { return zingers[i].CreatedAt.UTC().After(zingers[j].CreatedAt.UTC())})
	}

	type returnVals struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}

	jsonZingers := make([]returnVals, len(zingers))

	for i, zinger := range zingers {
		jsonZingers[i] = returnVals{
			Id:        zinger.ID,
			CreatedAt: zinger.CreatedAt,
			UpdatedAt: zinger.UpdatedAt,
			Body:      zinger.Body,
			UserId:    zinger.UserID,
		}
	}
	fmt.Println("AUTHOR:" + authorIDParam)
	fmt.Printf("zingers:")
	fmt.Println(zingers)
	fmt.Println("jsonZingers:")
	fmt.Println(jsonZingers)


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonZingers)
}



func (cfg *apiConfig) zingerGetHandler(w http.ResponseWriter, r *http.Request) {
	zingerID, err := uuid.Parse(r.PathValue("zingerID"))
	if err != nil {
		handleError(w, r, err)
		return
	}

	zinger, err := cfg.dbq.GetZingerById(context.Background(), zingerID)
	if err!= nil {
		handleErrorNotFound(w)
		return
	}

	type returnVals struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	
	respBody := returnVals{ID: zinger.ID, CreatedAt: zinger.CreatedAt, UpdatedAt: zinger.UpdatedAt, Body: zinger.Body, UserId: zinger.UserID}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(respBody)
	if err != nil {
		handleError(w, r, err)
		return
	}
	w.Write(dat)
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


func handleUserLoginError(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Error string `json:"error"`
	}
	respBody := returnVals{Error: "incorrect email or password"}
	dat, err := json.Marshal(respBody)
	if err!= nil {
		handleError(w, r, err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(401)
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


func handleErrorForbidden(w http.ResponseWriter) {
	w.WriteHeader(403)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("403 Forbidden"))
}

func handleErrorUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(401)
	w.Write([]byte("Unauthorized"))
}

func handleErrorNotFound(w http.ResponseWriter) {
	w.WriteHeader(404)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Not Found"))
}


func (cfg *apiConfig) zingersDeleteHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		handleErrorUnauthorized(w)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwt_secret)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidJWT) {
			handleErrorUnauthorized(w)
		} else {
			handleError(w, r, err)
		}
		return
	}
	zingerID, err := uuid.Parse(r.PathValue("zingerID"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	zinger, err := cfg.dbq.GetZingerById(context.Background(), zingerID)
	if err != nil {
		handleErrorNotFound(w)
		return
	}
	if zinger.UserID != userID {
		handleErrorForbidden(w)
		return
	}
	err = cfg.dbq.DeleteZingerById(context.Background(), zingerID)
	if err != nil {
		handleError(w, r, err)
		return
	}
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


func main() {

	serverHandler := http.NewServeMux()
	server := &http.Server{Addr: ":8080", Handler: serverHandler}
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	apiCfg := apiConfig{dbq: database.New(db), jwt_secret: os.Getenv("JWT_SECRET"), zingpay_key: os.Getenv("ZINGPAY_KEY")}

	serverHandler.Handle("/app/", apiCfg.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serverHandler.HandleFunc("GET /api/healthz", handler)
	serverHandler.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	serverHandler.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	serverHandler.HandleFunc("POST /api/users", apiCfg.postUsersHandler)
	serverHandler.HandleFunc("POST /api/zingers", apiCfg.zingersPostHandler)
	serverHandler.HandleFunc("POST /api/login", apiCfg.userLoginHandler)
	serverHandler.HandleFunc("GET /api/zingers", apiCfg.zingersGetHandler)
	serverHandler.HandleFunc("GET /api/zingers/{zingerID}", apiCfg.zingerGetHandler)
	serverHandler.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	serverHandler.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	serverHandler.HandleFunc("PUT /api/users", apiCfg.putUsersHandler)
	serverHandler.HandleFunc("DELETE /api/zingers/{zingerID}", apiCfg.zingersDeleteHandler)
	serverHandler.HandleFunc("POST /api/zingpay/webhooks", apiCfg.webhookHandler)




	fmt.Println("Starting server on :8080")
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}


