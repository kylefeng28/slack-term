package service

import (
	"database/sql"
	"os"
	fp "path/filepath"
	"time"

	"github.com/OpenPeeDeeP/xdg"
	_ "github.com/mattn/go-sqlite3"
)

type UserCache struct {
	db *sql.DB
}

func NewUserCache() (*UserCache, error) {
	cacheDir := fp.Join(xdg.CacheHome(), "slack-term",)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	dbPath := fp.Join(cacheDir, "users.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &UserCache{db: db}, nil
}

func (c *UserCache) Get(userID string) (string, bool) {
	var username string
	var updatedAt int64

	err := c.db.QueryRow(
		"SELECT username, updated_at FROM users WHERE user_id = ?",
		userID,
	).Scan(&username, &updatedAt)

	if err != nil {
		return "", false
	}

	// Cache expires after 7 days
	if time.Now().Unix()-updatedAt > 7*24*60*60 {
		return "", false
	}

	return username, true
}

func (c *UserCache) Set(userID, username string) error {
	_, err := c.db.Exec(
		"INSERT OR REPLACE INTO users (user_id, username, updated_at) VALUES (?, ?, ?)",
		userID, username, time.Now().Unix(),
	)
	return err
}

func (c *UserCache) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
