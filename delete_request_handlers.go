package main

import (
	"net/http"
	"github.com/bsuvonov/zingzing/internal/auth"
	"github.com/google/uuid"
	"errors"
	"context"
)


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