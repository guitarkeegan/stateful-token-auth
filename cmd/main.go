package main

import (
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	bolt "go.etcd.io/bbolt"
	stdlog "log"
)

type application struct {
	db     *bolt.DB
	logger *log.Logger
}

type logAdapter struct {
	logger *log.Logger
}

func (a *logAdapter) Write(p []byte) (n int, err error) {
	a.logger.Info(string(p))
	return len(p), nil
}

func main() {

	logger := log.New(os.Stdout)
	db := initDB()
	defer db.Close()

	app := &application{
		db:     db,
		logger: logger,
	}

	adapter := &logAdapter{logger: logger}
	serverLogger := stdlog.New(adapter, "", 0)

	server := http.Server{
		Addr:     ":8080",
		Handler:  app.routes(),
		ErrorLog: serverLogger,
	}

	logger.Info("Server running on port :8080")
	server.ListenAndServe()

}

func initDB() *bolt.DB {

	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatalf("db failed to open: %q", err)
	}

	return db
}
