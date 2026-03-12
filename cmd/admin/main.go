package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Ankater/last-1000/internal/db"
	"github.com/Ankater/last-1000/internal/tokens"
)

func main() {
	conn, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	repo := tokens.Repository{DB: conn}

	token, err := tokens.GenerateToken(32)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.CreateToken(ctx, token); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Token:")
	fmt.Println(token)
}
