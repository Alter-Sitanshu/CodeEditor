package store

import (
	"context"
	"database/sql"
	"time"
)

type RoomStore struct {
	db *sql.DB
}

type Room struct {
	Id        int64     `json:"id"`
	Name      string    `json:"name"`
	Author    *User     `json:"author"`
	Language  string    `json:"lang"`
	CreatedAt time.Time `json:"created_at"`
}

type RoomResponse struct {
	Room
	Members []Member `json:"members"`
}

type Member struct {
	Id      int64     `json:"id"`
	Name    string    `json:"name"`
	Role    string    `json:"role"`
	AddedAt time.Time `json:"added_at"`
}

func (r *RoomStore) Create(ctx context.Context, room *Room) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeOut)
	defer cancel()

	return withTx(r.db, ctx, func(tx *sql.Tx) error {
		query := `
			INSERT INTO rooms (name, author, language)
			VALUES ($1, $2, $3) RETURNING id, created_at
		`
		err := tx.QueryRowContext(ctx, query,
			room.Name,
			room.Author.Id,
			room.Language,
		).Scan(
			&room.Id,
			&room.CreatedAt,
		)

		if err != nil {
			return err
		}

		query = `
			INSERT INTO room_users (room_id, user_id, roleid)
			VALUES ($1, $2, $3)
		`
		_, err = tx.ExecContext(ctx, query, room.Id, room.Author.Id, 3)
		if err != nil {
			return err
		}

		return nil
	})

}

func (r *RoomStore) GetUserRooms(ctx context.Context, user *User) ([]Room, error) {
	query := `
		SELECT r.id, r.name, r.language, r.author, u.fname, u.lname
		FROM room_users ru
		JOIN rooms r ON r.id = ru.room_id
		JOIN users u ON u.id = r.author
		WHERE ru.user_id = $1
	`
	var output []Room
	rows, err := r.db.QueryContext(ctx, query, user.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ro Room
		ro.Author = &User{}
		err := rows.Scan(&ro.Id, &ro.Name, &ro.Language,
			&ro.Author.Id, &ro.Author.FirstName, &ro.Author.LastName,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, ro)
	}

	return output, nil

}

func (r *RoomStore) GetRoomById(ctx context.Context, roomID int64) (*RoomResponse, error) {
	query := `
		SELECT id, name, language, author
		FROM rooms
		WHERE id = $1
	`
	roomresp := &RoomResponse{}
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(
		&roomresp.Id,
		&roomresp.Name,
		&roomresp.Language,
		&roomresp.Author,
	)
	if err != nil {
		return nil, err
	}

	members, err := r.getMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	roomresp.Members = members

	return roomresp, nil
}

func (r *RoomStore) getMembers(ctx context.Context, roomID int64) ([]Member, error) {
	var members []Member
	query := `
		SELECT u.id, u.name, role, added_at
		FROM room_users
		JOIN users u ON u.id = user_id
		WHERE room_id = $1 AND u.active = TRUE
	`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return members, nil
		default:
			return nil, err
		}
	}
	defer rows.Close()
	for rows.Next() {
		var m Member
		err := rows.Scan(&m.Id, &m.Name, &m.Role, &m.AddedAt)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil

}
