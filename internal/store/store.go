package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const QueryTimeOut time.Duration = time.Minute * 3

var (
	ErrDupliMail    = errors.New("mail already exists")
	ErrTokenExpired = errors.New("token has expired")
)

type Storage struct {
	UserStore interface {
		create(context.Context, *sql.Tx, *User) error
		CreateAndInvite(context.Context, *User, string, time.Duration) error
		DeleteUser(context.Context, *User) error
		authorise(context.Context, *sql.Tx, string, time.Time) (*UserFromToken, error)
		ActivateUser(context.Context, string, time.Time) error
	}
}

func Mount(addr string, MaxConns, MaxIdleConns, MaxIdleTime int) (*sql.DB, error) {
	db, err := sql.Open("postgres", addr)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(time.Duration(MaxIdleTime) * time.Minute)
	db.SetMaxIdleConns(MaxIdleConns)
	db.SetMaxOpenConns(MaxConns)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func NewPostgresStore(db *sql.DB) Storage {
	return Storage{
		UserStore: &UserStore{
			db: db,
		},
	}
}

func withTx(db *sql.DB, ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
