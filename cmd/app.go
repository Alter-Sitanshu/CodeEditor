package main

import (
	"log"
	"time"

	"github.com/Alter-Sitanshu/CodeEditor/internal/auth"
	"github.com/Alter-Sitanshu/CodeEditor/internal/env"
	"github.com/Alter-Sitanshu/CodeEditor/internal/store"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env", err.Error())
	}

	cfg := Config{
		addr: env.GetString("PORT", ":8080"),
		dbcfg: DBConfig{
			addr:         env.GetString("DB_ADDR", "postgres://postgres:password@localhost:5432/editor?sslmode=disable"),
			MaxConns:     10,
			MaxIdleConns: 5,
			MaxIdleTime:  5,
		},
		tokencfg: TokenConfig{
			expiry: time.Hour * 3,
		},
	}

	authenticator := auth.NewAuthenticator(
		env.GetString("APP_SECRET", "secret"),
		env.GetString("APP_AUD", "GoCode"),
		env.GetString("APP_ISS", "GoCode"),
	)
	db, err := store.Mount(cfg.dbcfg.addr,
		cfg.dbcfg.MaxConns,
		cfg.dbcfg.MaxIdleConns,
		cfg.dbcfg.MaxIdleTime,
	)
	if err != nil {
		log.Println(err.Error())
		return
	}

	psql := store.NewPostgresStore(db)
	app := &Application{
		config:        cfg,
		database:      psql,
		authenticator: *authenticator,
	}

	handlerMux := app.mount()
	err = app.run(handlerMux)

	if err != nil {
		log.Println(err.Error())
	}
}
