package main

import (
	"fmt"
	"net/http"
	"strconv"
)

type Settings struct {
	Port int
}

func healthz(s *Settings) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "I am alive!\n")
	}
}

func content(s *Settings) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Lorem ipsum dolor...\n")
	}
}

func startServer(s *Settings) {
	http.HandleFunc("/healthz", healthz(s))
	http.HandleFunc("/contetn", content(s))

	http.ListenAndServe(":"+strconv.Itoa(s.Port), nil)
}

func main() {
	startServer(&Settings{})
}
