package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type db struct {
	conn *sql.DB
	host string
	port string
	user string
	pass string
	name string
}

func (d *db) GetConn() (*sql.DB, error) {
	sqlDB, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", d.host, d.port, d.user, d.pass, d.name))
	if err != nil {
		return nil, fmt.Errorf("error with sqlDB.Open in db.Connect: %s", err)
	}
	sqlDB.SetConnMaxLifetime(time.Minute * 3)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(30)

	err = sqlDB.Ping()
	if err != nil {
		return nil, fmt.Errorf("error with sqlDB.Ping in db.Connect: %s", err)
	}

	return sqlDB, nil
}

func NewDB(cfg *Config) *db {
	return &db{
		conn: nil,
		host: cfg.DBHost,
		port: cfg.DBPort,
		user: cfg.DBUser,
		pass: cfg.DBPass,
		name: cfg.DBName,
	}
}
