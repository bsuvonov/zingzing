package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"

	"github.com/bsuvonov/zingzing/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)


type apiConfig struct {
	fileserverHits atomic.Int32
	dbq *database.Queries
	jwt_secret string
	zingpay_key string
}



func (cfg *apiConfig) middlewareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
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
	serverHandler.HandleFunc("GET /api/healthz", handlerHealthz)
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


