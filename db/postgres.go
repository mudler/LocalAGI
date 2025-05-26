package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
)

var Conn *pgx.Conn

func ConnectDB(dbURL string) {
	var err error
	Conn, err = pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
}
