package srv

import "database/sql"

type Servr interface {
	Authenticate(username string, password string) error
	GetLoginAndDomain(email string) (string, string, error)
	IsLocalDomain(domain string) (bool, error)
	CheckLocalUserAndDomain(login string, domain string) error
	DeliverLocally(to string, body []byte) error
	DeliverQueue(from string, to string, bytes []byte) error
}

type server struct {
	db *sql.DB
}

func (s *server) DeliverLocally(to string, body []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s *server) DeliverQueue(from string, to string, bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s *server) CheckLocalUserAndDomain(login string, domain string) error {
	//TODO implement me
	panic("implement me")
}

func (s *server) IsLocalDomain(domain string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (s *server) GetLoginAndDomain(email string) (string, string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *server) Authenticate(username string, password string) error {
	//TODO implement me
	panic("implement me")
}

func NewServer(db *sql.DB) Servr {
	return &server{db: db}
}
