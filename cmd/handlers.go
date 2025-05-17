package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

type SuccessMsg struct {
	Msg string `json:"msg"`
}

func (app *application) handleIndex(w http.ResponseWriter, r *http.Request) {

	type WelcomeMsg struct {
		Msg string `json:"msg"`
	}

	msg := WelcomeMsg{Msg: "Come with me if you want to live."}

	err := app.sendJSON(w, http.StatusOK, &msg)
	if err != nil {
		app.logger.Errorf("handleIndex: %q", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
	}
}

func (app *application) handleCreate(w http.ResponseWriter, r *http.Request) {

	type Creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	cred := &Creds{}
	err := app.readJSON(r, cred)
	if err != nil {
		app.logger.Error("handleCreate", "error", err)
	}
	if cred.Password == "" || cred.Email == "" {
		http.Error(w, "no good", http.StatusUnauthorized)
		return
	}

	app.logger.Info("Create new user request", "email", cred.Email, "password", "REDACTED")

	// TODO: create new user with email and hash
	hash, err := bcrypt.GenerateFromPassword([]byte(cred.Password), 10)
	if err != nil {
		app.logger.Errorf("handleCreate: %q", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}

	err = app.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		err := b.Put([]byte(cred.Email), hash)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		app.logger.Errorf("handleCreate: %q", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
	}

	// setup session
	app.sessionManager.Put(r.Context(), "authenticated", true)

	sm := SuccessMsg{Msg: "New user created."}
	err = app.sendJSON(w, http.StatusCreated, sm)
	if err != nil {
		app.logger.Errorf("handleCreate: %q", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
	}

}

func (app *application) handleSignIn(w http.ResponseWriter, r *http.Request) {
}
func (app *application) handleSignout(w http.ResponseWriter, r *http.Request) {}
func (app *application) handleSecret(w http.ResponseWriter, r *http.Request) {

	msg := SuccessMsg{
		Msg: "Here are the secrets!",
	}

	app.sendJSON(w, http.StatusOK, msg)
}

func (app *application) sendJSON(w http.ResponseWriter, statusCode int, data any) error {
	env := map[string]any{
		"skynet": data,
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(env)
	if err != nil {
		return err
	}

	return nil
}

func (app *application) readJSON(r *http.Request, dst any) error {
	// Check if the Content-Type is application/json
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		return fmt.Errorf("content type header is not application/json")
	}

	// Limit the size of the request body to 1MB
	r.Body = http.MaxBytesReader(nil, r.Body, 1_048_576) // 1MB limit

	// Initialize the decoder with DisallowUnknownFields to catch typos and extra fields
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the request body into the destination struct
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return fmt.Errorf("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown field %s", fieldName)

		case errors.As(err, &invalidUnmarshalError):
			// This error indicates a programmer error, not a client error
			panic(err)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than 1MB")

		default:
			return fmt.Errorf("error parsing JSON: %v", err)
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return fmt.Errorf("body must only contain a single JSON value")
	}

	return nil
}
