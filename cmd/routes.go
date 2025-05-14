package main

import "net/http"

func (app *application) routes() *http.ServeMux {

	mux := http.NewServeMux()

	mux.Handle("GET /", http.HandlerFunc(app.handleIndex))
	mux.Handle("POST /api/create", http.HandlerFunc(app.handleCreate))
	mux.Handle("POST /api/signin", http.HandlerFunc(app.handleSignIn))
	mux.Handle("POST /api/signout", http.HandlerFunc(app.handleSignout))
	mux.Handle("GET /api/secret", http.HandlerFunc(app.handleSecret))

	return mux
}
