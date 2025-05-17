package main

import (
	"net/http"
	"os"
	"time"

	stdlog "log"

	"github.com/alexedwards/scs/boltstore"
	"github.com/alexedwards/scs/v2"
	"github.com/charmbracelet/log"
	bolt "go.etcd.io/bbolt"
)

type application struct {
	db             *bolt.DB
	logger         *log.Logger
	sessionManager *scs.SessionManager
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

	sm := scs.New()
	sm.Store = boltstore.NewWithCleanupInterval(db, 20*time.Second)
	sm.Lifetime = time.Minute * 5

	app := &application{
		db:             db,
		logger:         logger,
		sessionManager: sm,
	}

	adapter := &logAdapter{logger: logger}
	serverLogger := stdlog.New(adapter, "", 0)

	server := http.Server{
		Addr:     ":8080",
		Handler:  app.sessionManager.LoadAndSave(app.routes()),
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

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("sessons"))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatalf("bucket creation failed: %q", err)
	}

	return db
}
