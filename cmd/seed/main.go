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

	fmt.Println("Seeding payment gateways...")
	_, err = conn.Exec(ctx, `
		INSERT INTO public.payment_gateways (gateway_id, name, is_active) 
		VALUES (1, 'Stripe Simulation', true) 
		ON CONFLICT (gateway_id) DO NOTHING
	`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[OK] Payment gateway seeded.")
}
