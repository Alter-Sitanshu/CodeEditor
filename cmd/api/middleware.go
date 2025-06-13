package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type Userctx string
type Roomctx string

const userctx Userctx = "user"
const roomctx Roomctx = "room"

func (app *Application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Autherization")
		if authHeader == "" {
			jsonResponse(w, http.StatusUnauthorized, "auth header absent")
			return
		}
		HeaderVal := strings.Split(authHeader, " ")
		if len(HeaderVal) != 2 || HeaderVal[0] != "Bearer" {
			jsonResponse(w, http.StatusUnauthorized, "header malformed")
			return
		}
		JWTtoken, err := app.authenticator.ValidateToken(HeaderVal[1])
		if err != nil {
			jsonResponse(w, http.StatusUnauthorized, "header malformed")
			return
		}
		TokenSub, _ := JWTtoken.Claims.(jwt.MapClaims).GetSubject()
		userId, err := strconv.ParseInt(TokenSub, 10, 64)
		if err != nil {
			jsonResponse(w, http.StatusUnauthorized, "unauthorised")
			return
		}
		ctx := r.Context()
		user, err := app.database.UserStore.GetUserById(ctx, userId)
		if err != nil {
			jsonResponse(w, http.StatusUnauthorized, "unauthorised")
			return
		}
		ctx = context.WithValue(ctx, userctx, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *Application) BasicAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the auth header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
				jsonResponse(w, http.StatusUnauthorized, "auth header absent")
				return
			}
			// parse it
			authdata := strings.Split(authHeader, " ")
			if len(authdata) != 2 || authdata[0] != "Basic" {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
				jsonResponse(w, http.StatusUnauthorized, "auth header malformed")
				return
			}
			// decode it and check the creds
			decode, err := base64.StdEncoding.DecodeString(authdata[1])
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
				jsonResponse(w, http.StatusUnauthorized, err)
				return
			}
			creds := strings.SplitN(string(decode), ":", 2)
			username := app.config.auth.username
			pass := app.config.auth.pass

			if len(creds) != 2 || creds[0] != username || creds[1] != pass {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
				jsonResponse(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			// serve the route
			next.ServeHTTP(w, r)
		})
	}
}

// Room middleware
func (app *Application) RoomMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roomIdParam := chi.URLParam(r, "id")
		roomId, err := strconv.ParseInt(roomIdParam, 10, 64)
		ctx := r.Context()
		if err != nil {
			log.Println(err.Error())
			jsonResponse(w, http.StatusBadRequest, err.Error())
		}

		room, err := app.database.RoomStore.GetRoomById(ctx, roomId)
		if err != nil {
			log.Println(err.Error())
			jsonResponse(w, http.StatusInternalServerError, err.Error())
		}

		ctx = context.WithValue(ctx, roomctx, room)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
