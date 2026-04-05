package main

import (
	"context"
	"fmt"
	"log"

	"clippy/internal/db"
)

func main() {
	ctx := context.Background()

	database, err := db.Open(ctx, "./test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	fmt.Println("db opened successfully")
}
