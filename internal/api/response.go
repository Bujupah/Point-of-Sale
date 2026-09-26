package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"pos/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("api: failed to encode response: %v", err)
	}
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// writeError maps any error into the stable {"error":{code,message}} shape.
// Internal errors are logged with full detail but never leak past "Internal
// server error" to the client.
func writeError(w http.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		status := appErr.Status
		if status == 0 {
			status = 400
		}
		body := errorBody{}
		body.Error.Code = appErr.Code
		body.Error.Message = appErr.Message
		writeJSON(w, status, body)
		return
	}
	log.Printf("api: internal error: %v", err)
	body := errorBody{}
	body.Error.Code = "INTERNAL_ERROR"
	body.Error.Message = "Internal server error"
	writeJSON(w, http.StatusInternalServerError, body)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return domain.NewError("INVALID_INPUT", "Malformed request body", 400)
	}
	return nil
}
