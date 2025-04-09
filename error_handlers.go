package main

import (
	"net/http"
	"encoding/json"
	"log"
)

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

