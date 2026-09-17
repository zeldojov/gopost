package main

import (
	"database/sql"
	"log"
)

var LOG = log.Default()
var DB *sql.DB
var err error

func init() {
	if err = InitDB(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
}

func main() {
	defer DB.Close()

}
