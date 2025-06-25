package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func initDB() (*sql.DB, error) {
	host := "localhost"
	port := "5432"
	user := "postgres"
	password := "1809"
	dbname := "projeto_saude"

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar ao banco de dados: %v", err)
	}

	// Configurar conexão
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Testar conexão
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("erro ao testar conexão com o banco de dados: %v", err)
	}

	log.Println("Conexão com o banco de dados estabelecida com sucesso")
	return db, nil
}
