package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ankater/last-1000/internal/db"
	"github.com/Ankater/last-1000/internal/tokens"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) < 2 {
		log.Fatal("expected subcommand: create-token")
	}

	switch os.Args[1] {
	case "create-token":
		if err := runCreateToken(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown subcommand: %s", os.Args[1])
	}
}

func runCreateToken(args []string) error {
	createTokenCmd := flag.NewFlagSet("create-token", flag.ContinueOnError)
	name := createTokenCmd.String("name", "", "token name")
	if err := createTokenCmd.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("name is required")
	}

	conn, err := db.Open()
	if err != nil {
		return err
	}
	defer conn.Close()

	repo := tokens.Repository{DB: conn}
	service := tokens.Service{Repo: &repo}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	token, err := service.CreateToken(ctx, *name)
	if err != nil {
		return err
	}

	fmt.Println("Token:")
	fmt.Println(token)
	return nil
}
