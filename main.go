package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

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
}


func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	val := int(cfg.fileserverHits.Load())

	content := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
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


func censorChirp(input string) string {
	// Regex for "sharbert" (case-insensitive, optional ! at the end)
	sharbertRe := regexp.MustCompile(`(?i)\bsharbert\b!?`)

	// Regex for "kerfuffle" and "fornax" (case-insensitive, word boundaries only)
	kerfuffleRe := regexp.MustCompile(`(?i)\bkerfuffle\b`)
	fornaxRe := regexp.MustCompile(`(?i)\bfornax\b`)

	// Handle "sharbert" with exception for '!'
	output := sharbertRe.ReplaceAllStringFunc(input, func(match string) string {
		if strings.HasSuffix(match, "!") {
			return match
		}
		return "****"
	})

	output = kerfuffleRe.ReplaceAllStringFunc(output, func(match string) string {
		return "****"
	})

	output = fornaxRe.ReplaceAllStringFunc(output, func(match string) string {
		return "****"
	})

	return output
	}


func handleError(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Println(err.Error())
	w.WriteHeader(500)
	w.Header().Set("Content-Type", "application/json")

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
	w.Write(dat)
}





func (cfg *apiConfig) chirpsPostHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
		UserId string `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
		return
	}
	if len(params.Body) > 140 {
		handleError(w, r, err)
		return
	}
	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")

	params.Body = censorChirp(params.Body)

	type returnVals struct {
		Id uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body string `json:"body"`
		UserId string `json:"user_id"`
	}

	respBody := returnVals{
		Id: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body: params.Body,
		UserId: params.UserId,
	}

	cfg.dbq.CreateChirp(context.Background(), database.CreateChirpParams{ID: respBody.Id, CreatedAt: respBody.CreatedAt, UpdatedAt: respBody.UpdatedAt, Body: respBody.Body, UserID: respBody.UserId})

	dat, err := json.Marshal(respBody)
	if err!=nil {
		log.Printf("Error in converting response body to json: %s", err)
		return
	}
	w.Write(dat)
}


func (cfg *apiConfig) chirpsGetHandler(w http.ResponseWriter, r *http.Request) {
	
	chirps, err := cfg.dbq.GetAllChirps(context.Background())

	if err != nil {
		handleError(w, r, err)
        return
    }
    
    // Return the chirps as JSON with 200 status code
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(chirps)
}




func (cfg *apiConfig) usersHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		handleError(w, r, err)
		return
	}


	type returnVals struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	respBody := returnVals{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email: params.Email,
	}

	_, err = cfg.dbq.CreateUser(context.Background(), database.CreateUserParams{ID: respBody.ID, CreatedAt: respBody.CreatedAt, UpdatedAt: respBody.UpdatedAt, Email: respBody.Email})
	if err != nil {
		handleError(w, r, err)
		return
	}

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")

	dat, err := json.Marshal(respBody)
	if err!=nil {
		log.Printf("Error in converting response body to json: %s", err)
		return
	}
	w.Write(dat)
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

	apiCfg := apiConfig{dbq: database.New(db)}

	serverHandler.Handle("/app/", apiCfg.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serverHandler.HandleFunc("GET /api/healthz", handler)
	serverHandler.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	serverHandler.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	serverHandler.HandleFunc("POST /api/users", apiCfg.usersHandler)
	serverHandler.HandleFunc("POST /api/chirps", apiCfg.chirpsPostHandler)
	serverHandler.HandleFunc("GET /api/chirps", apiCfg.chirpsGetHandler)




	fmt.Println("Starting server on :8080")
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}


