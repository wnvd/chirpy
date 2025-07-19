package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// TODO: you probably need to split it into
// error response and json response
// helper function to send error response
func errResponseHandle(respType ResponseError, respMsg string, w http.ResponseWriter, _ *http.Request) {
	type errorResponse struct {
		Value string `json:"value"`
	}
	if respType == ServerError {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if respType == Rejected {
		w.WriteHeader(http.StatusBadRequest)
	}
	if respType == NotFound {
		w.WriteHeader(http.StatusNotFound)
	}
	if respType == Unauthorized {
		w.WriteHeader(http.StatusUnauthorized)
	}
	errResp := errorResponse{
		Value: respMsg,
	}

	response, err := json.Marshal(errResp)
	if err != nil {
		log.Printf("Failed to marshal json error response!")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.Write([]byte(response))
}

type ErrorMessage struct {
	Value string `json:"value"`
}

func ErrorResponse(w http.ResponseWriter, header int, msg string) {
	JSONResponse(w, header, ErrorMessage{Value: msg})
}

func JSONResponse(w http.ResponseWriter, header int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(header)
	res, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Unable to marshal error response %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(res))
}
