package main

import (
	"log"

	"github.com/Vill785/go_final_project/internal/db"
	"github.com/Vill785/go_final_project/internal/server"
)

func main() {
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer database.Close()

	srv := server.NewServer(database, ":7540", "web")
	if err := srv.Start(); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
