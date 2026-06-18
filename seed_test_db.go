package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		log.Fatal("TEST_DATABASE_URL environment variable is not set in .env")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	fmt.Println("Starting Seeding Process for Test Database...")

	// 1. Clean up existing test-specific data (handling dependencies)
	fmt.Println("Cleaning up previous test data...")
	tablesToTruncate := []string{
		"payments", 
		"contract_status_history",
		"contracts", 
		"property_status_history", 
		"property_events", 
		"visit_status_history", 
		"visits", 
		"transaction_status_history",
		"transactions",
	}
	for _, table := range tablesToTruncate {
		_, err := conn.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			fmt.Printf("Warning: Could not clean table %s: %v\n", table, err)
		}
	}

    // Clean up test properties, users and schedules
    conn.Exec(ctx, "DELETE FROM agent_schedules WHERE agent_id >= 200 AND agent_id <= 203")
    conn.Exec(ctx, "DELETE FROM residential_properties WHERE property_id >= 500")
    conn.Exec(ctx, "DELETE FROM locations WHERE property_id >= 500")
    conn.Exec(ctx, "DELETE FROM properties WHERE property_id >= 500")
    conn.Exec(ctx, "DELETE FROM users WHERE user_id >= 200 AND user_id <= 203")

	// 2. Insert Certified Test Users
	fmt.Println("Inserting Certified Test Users...")
	users := []struct {
		id    int
		uuid  string
		email string
		role  int
		first string
		last  string
	}{
		{200, "550e8400-e29b-41d4-a716-446655440000", "admin@spazio.com", 1, "Admin", "Test"},
		{201, "550e8400-e29b-41d4-a716-446655440001", "agent@spazio.com", 2, "Maria", "Agente"},
		{202, "550e8400-e29b-41d4-a716-446655440002", "owner@spazio.com", 1, "Carlos", "Dueño"},
		{203, "550e8400-e29b-41d4-a716-446655440003", "client@spazio.com", 3, "Juan", "Cliente"},
	}

	for _, u := range users {
		_, err := conn.Exec(ctx, `
			INSERT INTO users (user_id, user_uuid, email, role_id, first_name, last_name, phone, profile_picture_url, status_id)
			VALUES ($1, $2, $3, $4, $5, $6, '1234567890', '', 1)
			ON CONFLICT (user_id) DO UPDATE SET email = $3, role_id = $4`,
			u.id, u.uuid, u.email, u.role, u.first, u.last)
		if err != nil {
			log.Fatalf("Error inserting user %s: %v\n", u.email, err)
		}
	}

	// 3. Insert Certified Test Properties
	fmt.Println("Inserting Certified Test Properties...")
	// Property 500: Rent (Available)
	_, err = conn.Exec(ctx, `
		INSERT INTO properties (property_id, owner_id, agent_id, title, description, property_type_id, modality_id, status_id, cover_photo_url)
		VALUES (500, 202, 201, 'Casa Test Integration (Rent)', 'Casa de prueba para integración de rentas', 1, 2, 2, 'http://test.com/photo.jpg')`)
	if err != nil {
		log.Fatalf("Error inserting property 500: %v\n", err)
	}
    
    conn.Exec(ctx, `INSERT INTO residential_properties (property_id, bedrooms, bathrooms, floors, built_area) VALUES (500, 3, 2, 1, 120.5)`)
    conn.Exec(ctx, `INSERT INTO locations (property_id, city_id, street, exterior_number, neighborhood, latitude, longitude) 
                    VALUES (500, 1, 'Av. Universidad', '123', 'Centro', 19.4326, -99.1332)`)
    
    // Property 501: Sale (Available)
	_, err = conn.Exec(ctx, `
		INSERT INTO properties (property_id, owner_id, agent_id, title, description, property_type_id, modality_id, status_id, cover_photo_url)
		VALUES (501, 202, 201, 'Depto Test Integration (Sale)', 'Depto de prueba para integración de ventas', 2, 1, 2, 'http://test.com/photo2.jpg')`)
	if err != nil {
		log.Fatalf("Error inserting property 501: %v\n", err)
	}
    conn.Exec(ctx, `INSERT INTO residential_properties (property_id, bedrooms, bathrooms, floors, built_area) VALUES (501, 2, 1, 1, 85.0)`)
    conn.Exec(ctx, `INSERT INTO locations (property_id, city_id, street, exterior_number, neighborhood, latitude, longitude) 
                    VALUES (501, 1, 'Calle Reforma', '456', 'Juarez', 19.4271, -99.1677)`)

    // 4. Setup a Pending Transaction for Contract Testing
    fmt.Println("Setting up a Pending Transaction for Contract Testing...")
    _, err = conn.Exec(ctx, `
        INSERT INTO transactions (transaction_id, transaction_uuid, property_id, client_id, agent_id, transaction_type, status_id, final_amount, closing_date)
        VALUES (1000, '550e8400-e29b-41d4-a716-446655441000', 500, 203, 201, 'rent', 1, 5000.00, now() + interval '1 year')`)
    if err != nil {
        log.Fatalf("Error inserting transaction 1000: %v\n", err)
    }

    // 5. Setup an Active Contract for Payment Testing
    fmt.Println("Setting up an Active Contract for Payment Testing...")
    _, err = conn.Exec(ctx, `
        INSERT INTO contracts (contract_id, contract_uuid, transaction_id, period_id, currency, agreed_amount, storage_key, start_date, status_id)
        VALUES (1000, '550e8400-e29b-41d4-a716-446655442000', 1000, 3, 'MXN', 5000.00, 'contracts/test.pdf', now(), 2)`)
    if err != nil {
        log.Fatalf("Error inserting contract 1000: %v\n", err)
    }

    // 6. Setup Agent Schedule for Visit Testing (Maria Agente ID 201)
    fmt.Println("Setting up Agent Schedule and Property Assignment...")
    for day := 0; day <= 6; day++ {
        _, err = conn.Exec(ctx, `
            INSERT INTO agent_schedules (agent_id, day_of_week, start_time, end_time, is_active)
            VALUES (201, $1, '09:00:00', '18:00:00', true)`, day)
        if err != nil {
            log.Fatalf("Error inserting schedule for day %d: %v\n", day, err)
        }
    }

	fmt.Println("\nSeeding completed successfully! Database is ready for integration tests.")
    fmt.Println("- Admin: ID 200")
    fmt.Println("- Agent: ID 201")
    fmt.Println("- Client: ID 203")
    fmt.Println("- Property Rent: ID 500 (Available)")
    fmt.Println("- Pending Transaction: ID 1000 (Property 500, Client 203)")
}
