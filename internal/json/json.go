package json

import (
	"encoding/json"
	messages "my-personal-lib/internal/error"
	"net/http"
)

func Write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func Read(r *http.Request, data any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(data)
	if err != nil {
		return messages.ErrInvalidJSON
	}
	if decoder.More() {
		return messages.ErrDecode
	}
	return nil
}
