package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	var install bool

	// Получаем рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fmt.Println("Working directory:", wd)

	// Создаем папку "db" в рабочей директории
	dbDir := filepath.Join(wd, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dbDir, "scheduler.db")

	// Проверяем, существует ли файл базы данных
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		install = true
	} else if err != nil {
		return nil, err
	}

	// Открываем или создаем базу данных
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Если база данных новая, создаем таблицу и индекс
	if install {
		createTable := `
		CREATE TABLE IF NOT EXISTS scheduler (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL CHECK(length(date) = 8),
			title TEXT NOT NULL,
			comment TEXT,
			repeat VARCHAR(128)
		);`

		createIndex := `
		CREATE INDEX IF NOT EXISTS idx_date ON scheduler (date);`

		if _, err := database.Exec(createTable); err != nil {
			return nil, err
		}
		if _, err := database.Exec(createIndex); err != nil {
			return nil, err
		}
	}

	return database, nil
}
