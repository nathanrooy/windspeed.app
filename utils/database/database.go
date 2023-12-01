package database

import (
    "crypto/md5"
	"database/sql"
    "fmt"
	"log"
	"os"
    "time"

	_ "github.com/lib/pq"
)

const NewUser    string = `{"event_type":"user_subscribed"}`
const DeleteUser string = `{"event_type":"user_unsubscribed"}`

func connectionString() string {
	var url string = "postgresql://"
	url += os.Getenv("DB_USER") + ":"
	url += os.Getenv("DB_PSWD") + "@"
	url += os.Getenv("DB_HOST") + ":"
	url += os.Getenv("DB_PORT") + "/"
	url += os.Getenv("DB_DATABASE") + "?sslmode=verify-full"
	return url
}

func Connect() *sql.DB {
	db, err := sql.Open("postgres", connectionString())
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func AddEvent(userId string, service string, event string, db *sql.DB) {
    log.Printf("> adding event\n")
    anonymousId := CreateAnonymousId(fmt.Sprintf("%v", userId), service)
                                    
    sql := `INSERT INTO %s.events.events (event_time, anonymous_id, service, event) VALUES($1, $2, $3, $4);`
	stmt, err := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE")))
	if err != nil {
		log.Printf("> failed to prepare insert: %v", err)
	}
	result, err := stmt.Exec(time.Now().Unix(), anonymousId, service, event)
	if err != nil {
		log.Printf("db-err: %s\n", err)
	}
    rowsAffected, err := result.RowsAffected()
	if rowsAffected == 1 {
		log.Printf("successfully added event for: %v\n", anonymousId)
	} else {
		log.Printf("failed to add event for: %v\n", anonymousId)
	}
}

func CreateAnonymousId(userId string, service string) string {
    return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v-%v", userId, service))))
}
