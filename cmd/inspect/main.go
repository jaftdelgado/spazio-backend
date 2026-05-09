package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	databaseURL := "postgresql://neondb_owner:npg_imt3PQIZog6H@ep-twilight-king-amd0hxe1-pooler.c-5.us-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
	
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	defer conn.Close(ctx)

	fmt.Println("Checking columns in 'payments' table...")
	rows, err := conn.Query(ctx, `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_schema = 'public' AND table_name = 'payments'
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var columnName string
		err := rows.Scan(&columnName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Column: %s\n", columnName)
	}
}
