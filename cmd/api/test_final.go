package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaftdelgado/spazio-backend/internal/config"
	"github.com/jaftdelgado/spazio-backend/internal/modules/contracts"
	"github.com/jaftdelgado/spazio-backend/internal/storage"
)

func main() {
	dbURL := "postgresql://neondb_owner:npg_imt3PQIZog6H@ep-twilight-king-amd0hxe1-pooler.c-5.us-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
	
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("❌ Error conectando a DB: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 Buscando transacción válida en Neon...")
	var transactionID int32
	var ownerID int32
	// Buscamos cualquier transacción activa para probar
	err = db.QueryRow(ctx, `
		SELECT t.transaction_id, p.owner_id 
		FROM transactions t
		JOIN properties p ON t.property_id = p.property_id
		LIMIT 1
	`).Scan(&transactionID, &ownerID)

	if err != nil {
		log.Fatalf("❌ No se encontró una transacción para probar: %v", err)
	}
	fmt.Printf("✅ Transacción encontrada: %d (Propietario: %d)\n", transactionID, ownerID)

	// Configuración de R2 Real
	r2Cfg := config.R2Config{
		AccountID:       "219ffee0c80b6bdf473d8520eedc5de2",
		AccessKeyID:     "c1e36c50404ddfe187668855173c054a",
		SecretAccessKey: "49204b896a5d39472d0ec7931fbb14b2faa5c944319c2ccf4127608829c6f0c8",
		BucketName:      "spazio-storage",
	}

	r2Client, err := storage.NewR2Client(r2Cfg)
	if err != nil {
		log.Fatalf("❌ Error configurando R2: %v", err)
	}

	// Inicializar Módulo
	repo := contracts.NewRepository(db)
	svc := contracts.NewService(repo, r2Client)

	// Datos de prueba
	input := contracts.CreateContractInput{
		TransactionID: transactionID,
		Currency:      "MXN",
		AgreedAmount:  25000.00,
		StartDate:     time.Now(),
		EndDate:       nil,
	}

	fmt.Println("🚀 Generando contrato y subiendo a R2...")
	result, err := svc.GenerateContract(ctx, ownerID, input)
	if err != nil {
		log.Fatalf("❌ Error en GenerateContract: %v", err)
	}

	fmt.Println("\n✨ ¡PRUEBA EXITOSA!")
	fmt.Printf("📄 UUID del Contrato: %s\n", result.ContractUUID)
	fmt.Printf("☁️  Key en R2: %s\n", result.StorageKey)
	fmt.Println("--------------------------------------------------")
}
