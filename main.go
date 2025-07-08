package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	mux := http.NewServeMux()

	server := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Print("%w", err)
		os.Exit(1)
	}

}
