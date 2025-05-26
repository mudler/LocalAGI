package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Conn *pgxpool.Pool

func ConnectDB(dbURL string) {
	var err error
	Conn, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal("DB pool connection error:", err)
	}
}
