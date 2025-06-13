package main

import (
	"log"
	"time"

	"github.com/Alter-Sitanshu/CodeEditor/internal/auth"
	"github.com/Alter-Sitanshu/CodeEditor/internal/env"
	"github.com/Alter-Sitanshu/CodeEditor/internal/sockets"
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
		auth: BasicAuthConfig{
			username: env.GetString("ADMIN_USER", "admin"),
			pass:     env.GetString("ADMIN_PASS", "admin"),
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
		log.Fatal(err.Error())
	}
	defer db.Close()

	psql := store.NewPostgresStore(db)

	RoomHub := sockets.NewHub()
	executor := sockets.NewJudge0Executor()

	app := &Application{
		config:        cfg,
		database:      psql,
		authenticator: *authenticator,
		hub:           RoomHub,
		executor:      executor,
	}

	app.hub.Run()
	handlerMux := app.mount()
	err = app.run(handlerMux)

	if err != nil {
		log.Println(err.Error())
	}
}
