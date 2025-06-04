package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alter-Sitanshu/CodeEditor/internal/auth"
	"github.com/Alter-Sitanshu/CodeEditor/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Application struct {
	config        Config
	database      store.Storage
	authenticator auth.Authenticator
}

type Config struct {
	addr     string
	dbcfg    DBConfig
	tokencfg TokenConfig
	auth     BasicAuthConfig
}

type DBConfig struct {
	addr         string
	MaxConns     int
	MaxIdleConns int
	MaxIdleTime  int
}

type TokenConfig struct {
	expiry time.Duration
}

type BasicAuthConfig struct {
	username string
	pass     string
}

func (app *Application) mount() http.Handler {
	router := chi.NewRouter()
	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	router.Use(middleware.Timeout(60 * time.Second))

	router.Route("/v1", func(r chi.Router) {
		r.With(app.BasicAuthMiddleware()).Get("/health", app.HealthCheckHandler)

		// The sign in page will make a post request here to get the JWT
		r.Route("/auth", func(r chi.Router) {
			r.Post("/token", app.TokenHandler)
		})

		r.Route("/signup", func(r chi.Router) {
			r.Post("/", app.CreateUserHandler)
			r.Post("/activate", app.ActivateUserHandler)
		})

		r.Route("/user", func(r chi.Router) {
			r.Use(app.AuthMiddleware)
			r.Get("/", app.GetUserHandler)
		})

		r.Route("/rooms", func(r chi.Router) {
			r.Use(app.AuthMiddleware)
			r.Get("/", app.GetUserRoomsHandler)
			r.Post("/", app.CreateRoomHandler)
			r.Get("/{id}", app.GetRoomHandler)
		})
	})

	return router
}

func (app *Application) run(mux http.Handler) error {
	server := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	// Graceful shutdown
	shutdown := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Waits for the server closing signal
		s := <-quit
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()

		log.Printf("signal caught: %s", s.String())
		log.Println("Gracefully shutting down")
		shutdown <- server.Shutdown(ctx) // Sends the error as to server is closed
	}()
	fmt.Printf("Server listening at http://localhost%s\n", app.config.addr)
	err := server.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdown
	if err != nil {
		log.Println(err.Error())
		return err
	}

	log.Println("Server Shutdown successful")
	return nil
}
