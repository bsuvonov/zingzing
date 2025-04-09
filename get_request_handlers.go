package main


import (
	"net/http"
	"encoding/json"
	"github.com/google/uuid"
	"context"
	"github.com/bsuvonov/zingzing/internal/database"
	"time"
	"fmt"
	"sort"
)


func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
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


