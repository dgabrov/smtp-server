package srv

import "database/sql"

type Servr interface {
}

type server struct {
	db *sql.DB
}

func NewServer(db *sql.DB) Servr {
	return &server{db: db}
}
