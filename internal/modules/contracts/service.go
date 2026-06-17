package contracts

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaftdelgado/spazio-backend/internal/sqlcgen"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

const (
	roleAdminID = int32(1)
	roleAgentID = int32(2)

	transactionStatusClosedID = int32(3)
	propertyStatusSoldID      = int32(3)
)

type service struct {
	repository ContractRepository
	storage    ContractStorage
}

func NewService(repository ContractRepository, storage ContractStorage) ContractService {
	return &service{
		repository: repository,
		storage:    storage,
	}
}

func (s *service) ListContracts(ctx context.Context, userID int32, roleID int32, filter ListContractsFilter) ([]ContractListItem, error) {
	params := sqlcgen.ListContractsParams{
		Limit:  filter.Limit,
		Offset: (filter.Page - 1) * filter.Limit,
	}

	if roleID != roleAdminID && roleID != roleAgentID {
		params.FilterUserID = pgtype.Int4{Int32: userID, Valid: true}
	} else if filter.OwnerID != nil {
		params.FilterUserID = pgtype.Int4{Int32: *filter.OwnerID, Valid: true}
	}

	if filter.TransactionType != nil {
		params.TransactionType = sqlcgen.NullTransactionType{
			TransactionType: sqlcgen.TransactionType(*filter.TransactionType),
			Valid:           true,
		}
	}

	if filter.StatusID != nil {
		params.StatusID = pgtype.Int4{Int32: *filter.StatusID, Valid: true}
	}

	if filter.StartDate != nil {
		params.StartDate = pgtype.Timestamptz{Time: *filter.StartDate, Valid: true}
	}

	if filter.EndDate != nil {
		params.EndDate = pgtype.Timestamptz{Time: *filter.EndDate, Valid: true}
	}

	if filter.Search != nil {
		params.Search = pgtype.Text{String: *filter.Search, Valid: true}
	}

	rows, err := s.repository.ListContracts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}

	result := make([]ContractListItem, len(rows))
	for i, row := range rows {
		amount, _ := row.AgreedAmount.Float64Value()

		result[i] = ContractListItem{
			ContractUUID:    formatPgUUID(row.ContractUuid),
			TransactionType: string(row.TransactionType),
			PropertyTitle:   row.PropertyTitle,
			AgreedAmount:    amount.Float64,
			Currency:        row.Currency,
			StartDate:       row.StartDate.Time,
			Status:          row.StatusName,
			ClientName:      strings.TrimSpace(fmt.Sprintf("%v", row.ClientName)),
			CreatedAt:       row.CreatedAt.Time,
		}
	}

	return result, nil
}

func (s *service) GetContractDetail(ctx context.Context, userID int32, roleID int32, contractUUID uuid.UUID) (ContractDetail, error) {
	row, err := s.repository.GetContractByUUID(ctx, contractUUID)
	if err != nil {
		return ContractDetail{}, fmt.Errorf("get contract by uuid: %w", err)
	}

	clientHasAccess := row.ClientID != 0 && row.ClientID == userID

	if roleID != roleAdminID && roleID != roleAgentID && row.OwnerID != userID && !clientHasAccess {
		return ContractDetail{}, fmt.Errorf("no tiene permiso para ver este contrato")
	}

	amount, _ := row.AgreedAmount.Float64Value()

	pdfURL, err := s.storage.PublicURL(ctx, row.StorageKey)
	if err != nil {
		return ContractDetail{}, fmt.Errorf("generate contract public url: %w", err)
	}

	var endDate *time.Time
	if row.EndDate.Valid {
		endDate = &row.EndDate.Time
	}

	return ContractDetail{
		ContractID:    row.ContractID,
		ContractUUID:  formatPgUUID(row.ContractUuid),
		PropertyTitle: row.PropertyTitle,
		OwnerName:     fullName(row.OwnerFirstName, row.OwnerLastName),
		ClientName:    fullName(row.ClientFirstName, row.ClientLastName),
		AgreedAmount:  amount.Float64,
		Currency:      row.Currency,
		PeriodName:    row.PeriodName.String,
		StartDate:     row.StartDate.Time,
		EndDate:       endDate,
		Status:        row.StatusName,
		PDFUrl:        pdfURL,
	}, nil
}

func (s *service) GenerateRentContract(ctx context.Context, userID int32, input CreateRentContractInput) (CreateContractResult, error) {
	data, err := s.repository.GetContractDataByTransactionID(ctx, input.TransactionID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("fetch transaction data: %w", err)
	}

	if data.TransactionType != sqlcgen.TransactionTypeRent {
		return CreateContractResult{}, fmt.Errorf("la transacción no corresponde a una renta")
	}

	if data.ClientID == 0 || data.ClientID != userID {
		return CreateContractResult{}, fmt.Errorf("no tiene permiso para generar el contrato de esta renta")
	}

	internalInput := CreateContractInput{
		TransactionID: input.TransactionID,
		PeriodID:      &input.PeriodID,
		Currency:      input.Currency,
		AgreedAmount:  input.AgreedAmount,
		StartDate:     input.StartDate,
		EndDate:       &input.EndDate,
	}

	return s.createContractInternal(ctx, userID, internalInput, data)
}

func (s *service) GenerateSaleContract(ctx context.Context, userID int32, input CreateSaleContractInput) (CreateContractResult, error) {
	data, err := s.repository.GetContractDataByTransactionID(ctx, input.TransactionID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("fetch transaction data: %w", err)
	}

	if data.TransactionType != sqlcgen.TransactionTypeSale {
		return CreateContractResult{}, fmt.Errorf("la transacción no corresponde a una venta")
	}

	if data.AgentID != userID {
		return CreateContractResult{}, fmt.Errorf("sólo el agente asignado puede generar el contrato de esta venta")
	}

	saleDate := data.ClosingDate.Time

	internalInput := CreateContractInput{
		TransactionID: input.TransactionID,
		PeriodID:      nil,
		Currency:      input.Currency,
		AgreedAmount:  input.AgreedAmount,
		StartDate:     saleDate,
		EndDate:       nil,
	}

	return s.createContractInternal(ctx, userID, internalInput, data)
}

func (s *service) createContractInternal(ctx context.Context, userID int32, input CreateContractInput, data sqlcgen.GetContractDataByTransactionIDRow) (CreateContractResult, error) {
	if input.EndDate != nil && !input.EndDate.After(input.StartDate) {
		return CreateContractResult{}, fmt.Errorf("la fecha de finalización debe ser posterior a la fecha de inicio")
	}

	exists, err := s.repository.CheckContractExistsByTransactionID(ctx, input.TransactionID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("check contract existence: %w", err)
	}
	if exists {
		return CreateContractResult{}, fmt.Errorf("ya existe un contrato generado para esta transacción")
	}

	transactionAmount, _ := data.FinalAmount.Float64Value()
	if int64(input.AgreedAmount*100) != int64(transactionAmount.Float64*100) {
		return CreateContractResult{}, fmt.Errorf("el monto acordado (%.2f) no coincide con el monto de la transacción (%.2f)", input.AgreedAmount, transactionAmount.Float64)
	}

	clauses, err := s.repository.GetPropertyClausesByTransactionID(ctx, input.TransactionID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("fetch property clauses: %w", err)
	}

	amenities, err := s.repository.GetPropertyServicesByTransactionID(ctx, input.TransactionID)
	if err != nil {
		amenities = []string{}
	}

	var parentContractID *int32
	if data.TransactionType == sqlcgen.TransactionTypeRent && data.ClientID != 0 {
		if prevID, err := s.repository.FindLatestContractByPropertyAndClient(ctx, data.PropertyID, data.ClientID); err == nil && prevID > 0 {
			parentContractID = &prevID
		}
	}

	contractUUID := uuid.New()

	pdfBytes, err := s.generatePDF(data, clauses, amenities, input, contractUUID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("generate pdf: %w", err)
	}

	tx, err := s.repository.Begin(ctx)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	repoTx := s.repository.WithTx(tx)

	storageKey := fmt.Sprintf("contracts/%s.pdf", contractUUID.String())

	record, err := repoTx.CreateContract(ctx, contractUUID, input, parentContractID, storageKey)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("create contract record: %w", err)
	}

	err = s.storage.Upload(ctx, storageKey, "application/pdf", bytes.NewReader(pdfBytes))
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("upload pdf to storage: %w", err)
	}

	err = repoTx.UpdateTransactionStatus(ctx, input.TransactionID, transactionStatusClosedID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("update transaction status: %w", err)
	}

	if data.TransactionType == sqlcgen.TransactionTypeSale {
		err = repoTx.UpdatePropertyStatus(ctx, data.PropertyID, propertyStatusSoldID)
		if err != nil {
			return CreateContractResult{}, fmt.Errorf("update property status: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CreateContractResult{}, fmt.Errorf("commit transaction: %w", err)
	}

	pdfURL, err := s.storage.PublicURL(ctx, storageKey)
	if err != nil {
		pdfURL = ""
	}

	return CreateContractResult{
		ContractID:   record.ContractID,
		ContractUUID: contractUUID.String(),
		StorageKey:   storageKey,
		PDFUrl:       pdfURL,
	}, nil
}

func (s *service) generatePDF(data sqlcgen.GetContractDataByTransactionIDRow, clauses []sqlcgen.GetPropertyClausesByTransactionIDRow, amenities []string, input CreateContractInput, contractUUID uuid.UUID) ([]byte, error) {
	spanishMonths := map[string]string{
		"January": "Enero", "February": "Febrero", "March": "Marzo", "April": "Abril",
		"May": "Mayo", "June": "Junio", "July": "Julio", "August": "Agosto",
		"September": "Septiembre", "October": "Octubre", "November": "Noviembre", "December": "Diciembre",
	}

	formatDateSpanish := func(t time.Time) string {
		engMonth := t.Format("January")
		return fmt.Sprintf("%d de %s de %d", t.Day(), spanishMonths[engMonth], t.Year())
	}

	duration := "Indefinida"
	if input.EndDate != nil {
		days := int(input.EndDate.Sub(input.StartDate).Hours() / 24)
		if days >= 365 {
			duration = fmt.Sprintf("%d años", days/365)
		} else if days >= 30 {
			duration = fmt.Sprintf("%d meses", days/30)
		} else if days >= 7 {
			duration = fmt.Sprintf("%d semanas", days/7)
		} else {
			duration = fmt.Sprintf("%d días", days)
		}
	}

	title := "CONTRATO DE ARRENDAMIENTO"
	periodInfo := ""
	isRent := data.TransactionType == sqlcgen.TransactionTypeRent
	clientName := fullName(data.ClientFirstName, data.ClientLastName)

	if !isRent {
		title = "CONTRATO DE COMPRAVENTA"
	} else {
		pName := "Mensual"
		if input.PeriodID != nil {
			switch *input.PeriodID {
			case 1:
				pName = "Diaria"
			case 2:
				pName = "Semanal"
			case 3:
				pName = "Mensual"
			case 4:
				pName = "Anual"
			}
		} else if data.PeriodName.Valid {
			pName = translatePeriod(data.PeriodName.String)
		}

		periodInfo = fmt.Sprintf(" con frecuencia de pago %s", strings.ToLower(pName))
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(20).
		WithRightMargin(20).
		WithTopMargin(20).
		Build()

	m := maroto.New(cfg)

	m.AddRows(
		row.New(20).Add(
			col.New(12).Add(
				text.New(title, props.Text{
					Size:  18,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		),
		row.New(10).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Folio Digital: %s", contractUUID.String()), props.Text{
					Size:  8,
					Align: align.Right,
					Style: fontstyle.Italic,
				}),
			),
		),
		row.New(10).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Fecha de Emisión: %s", formatDateSpanish(time.Now())), props.Text{
					Size:  10,
					Align: align.Right,
				}),
			),
		),
		line.NewRow(5),
	)

	m.AddRows(
		row.New(12).Add(
			col.New(12).Add(
				text.New("I. DECLARACIONES", props.Text{Style: fontstyle.Bold, Size: 11}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("1.1 EL PROPIETARIO: %s %s, con plena capacidad legal para celebrar el presente contrato.", data.OwnerFirstName, data.OwnerLastName), props.Text{Size: 10}),
			),
		),
	)

	if isRent || clientName != "" {
		m.AddRows(
			row.New(8).Add(
				col.New(12).Add(
					text.New(fmt.Sprintf("1.2 EL CLIENTE: %s, quien manifiesta su interés en adquirir los derechos de uso sobre el inmueble.", clientName), props.Text{Size: 10}),
				),
			),
		)
	} else {
		m.AddRows(
			row.New(8).Add(
				col.New(12).Add(
					text.New("1.2 OPERACIÓN DE COMPRAVENTA: La venta fue formalizada por el agente responsable, sin cliente registrado en la transacción.", props.Text{Size: 10}),
				),
			),
		)
	}

	m.AddRows(row.New(10))

	m.AddRows(
		row.New(12).Add(
			col.New(12).Add(
				text.New("II. DEL INMUEBLE", props.Text{Style: fontstyle.Bold, Size: 11}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("2.1 UBICACIÓN: %s #%s, Col. %s, %s, %s.", data.Street, data.ExteriorNumber, data.Neighborhood, data.CityName, data.StateName), props.Text{Size: 10}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("2.2 DESCRIPCIÓN: %s", data.PropertyDescription), props.Text{Size: 10}),
			),
		),
	)

	lotArea, _ := data.LotArea.Float64Value()
	techInfo := fmt.Sprintf("Tipo: %s | Terreno: %.2f m2", data.PropertyTypeName, lotArea.Float64)
	if data.Bedrooms.Valid {
		builtArea, _ := data.BuiltArea.Float64Value()
		techInfo += fmt.Sprintf(" | Área Const.: %.2f m2 | Habitaciones: %d | Baños: %d | Pisos: %d", builtArea.Float64, data.Bedrooms.Int16, data.Bathrooms.Int16, data.Floors.Int16)
	}

	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New(techInfo, props.Text{Size: 9, Style: fontstyle.Italic}),
			),
		),
	)

	if len(amenities) > 0 {
		var translatedAmenities []string
		for _, a := range amenities {
			translatedAmenities = append(translatedAmenities, translateAmenity(a))
		}

		m.AddRows(
			row.New(10).Add(
				col.New(12).Add(
					text.New("2.3 SERVICIOS Y AMENIDADES INCLUIDOS:", props.Text{Style: fontstyle.Bold, Size: 10}),
				),
			),
			row.New(8).Add(
				col.New(12).Add(
					text.New(strings.Join(translatedAmenities, ", "), props.Text{Size: 9}),
				),
			),
		)
	}

	m.AddRows(row.New(10))

	objectText := "El propietario otorga al cliente la " + translateType(string(data.TransactionType)) + " del inmueble anteriormente descrito."
	if !isRent && clientName == "" {
		objectText = "El propietario formaliza la venta del inmueble anteriormente descrito mediante el agente responsable."
	}

	m.AddRows(
		row.New(12).Add(
			col.New(12).Add(
				text.New("III. CLÁUSULAS", props.Text{Style: fontstyle.Bold, Size: 11}),
			),
		),
		row.New(12).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("PRIMERA (Objeto): %s", objectText), props.Text{Size: 10}),
			),
		),
		row.New(12).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("SEGUNDA (Monto y Moneda): Las partes acuerdan un monto de %.2f %s%s, pagaderos íntegramente conforme a lo estipulado.", input.AgreedAmount, input.Currency, periodInfo), props.Text{Size: 10}),
			),
		),
		row.New(12).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("TERCERA (Vigencia): El presente instrumento legal surte efectos el %s, con una vigencia de %s.", formatDateSpanish(input.StartDate), duration), props.Text{Size: 10}),
			),
		),
	)

	if isRent && len(clauses) > 0 {
		m.AddRows(
			row.New(10).Add(
				col.New(12).Add(
					text.New("CUARTA (Condiciones Particulares):", props.Text{Style: fontstyle.Bold, Size: 10}),
				),
			),
		)

		clauseCount := 4
		for _, c := range clauses {
			phrase := ""

			switch c.ClauseName {
			case "pets allowed":
				if c.BooleanValue.Valid && c.BooleanValue.Bool {
					phrase = "Se permite la tenencia de mascotas en el inmueble."
				} else {
					phrase = "Queda estrictamente prohibida la tenencia de mascotas en el inmueble."
				}
			case "smoking allowed":
				if c.BooleanValue.Valid && c.BooleanValue.Bool {
					phrase = "Se permite fumar dentro del inmueble."
				} else {
					phrase = "Queda estrictamente prohibido fumar dentro del inmueble."
				}
			case "children allowed":
				if c.BooleanValue.Valid && c.BooleanValue.Bool {
					phrase = "El inmueble es apto y permite la residencia de menores de edad."
				} else {
					phrase = "No se permite la residencia de menores de edad en el inmueble."
				}
			default:
				switch c.ValueTypeCode {
				case "BOOLEAN":
					if c.BooleanValue.Valid && c.BooleanValue.Bool {
						phrase = fmt.Sprintf("Se permite: %s.", c.ClauseName)
					} else {
						phrase = fmt.Sprintf("No se permite: %s.", c.ClauseName)
					}
				case "INTEGER":
					if c.IntegerValue.Valid {
						phrase = fmt.Sprintf("Límite de %s: %d unidades.", c.ClauseName, c.IntegerValue.Int32)
					}
				case "RANGE":
					phrase = fmt.Sprintf("El rango permitido para %s es de %v a %v.", c.ClauseName, c.MinValue, c.MaxValue)
				}
			}

			if phrase != "" {
				m.AddRows(
					row.New(8).Add(
						col.New(12).Add(
							text.New(fmt.Sprintf("   %d.1 %s", clauseCount, phrase), props.Text{Size: 9}),
						),
					),
				)

				clauseCount++
			}
		}
	}

	secondSignatureLabel := "EL CLIENTE"
	secondSignatureName := clientName
	if !isRent && clientName == "" {
		secondSignatureLabel = "AGENTE RESPONSABLE"
		secondSignatureName = "Agente responsable"
	}

	m.AddRows(
		row.New(40),
		row.New(10).Add(
			col.New(5).Add(
				text.New("__________________________", props.Text{Align: align.Center}),
			),
			col.New(2),
			col.New(5).Add(
				text.New("__________________________", props.Text{Align: align.Center}),
			),
		),
		row.New(8).Add(
			col.New(5).Add(
				text.New("EL PROPIETARIO", props.Text{Align: align.Center, Size: 9, Style: fontstyle.Bold}),
			),
			col.New(2),
			col.New(5).Add(
				text.New(secondSignatureLabel, props.Text{Align: align.Center, Size: 9, Style: fontstyle.Bold}),
			),
		),
		row.New(8).Add(
			col.New(5).Add(
				text.New(fullName(data.OwnerFirstName, data.OwnerLastName), props.Text{Align: align.Center, Size: 9}),
			),
			col.New(2),
			col.New(5).Add(
				text.New(secondSignatureName, props.Text{Align: align.Center, Size: 9}),
			),
		),
	)

	document, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return document.GetBytes(), nil
}

func fullName(firstName string, lastName string) string {
	return strings.TrimSpace(firstName + " " + lastName)
}

func formatPgUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", value.Bytes[0:4], value.Bytes[4:6], value.Bytes[6:8], value.Bytes[8:10], value.Bytes[10:16])
}

func translatePeriod(p string) string {
	switch strings.ToLower(p) {
	case "daily", "diario":
		return "Diaria"
	case "weekly", "semanal":
		return "Semanal"
	case "yearly", "anual":
		return "Anual"
	default:
		return "Mensual"
	}
}

func translateType(t string) string {
	if t == "rent" {
		return "renta"
	}

	return "venta"
}

func translateAmenity(code string) string {
	switch strings.ToUpper(code) {
	case "POOL":
		return "Piscina"
	case "24H_SECURITY":
		return "Seguridad 24h"
	case "WIFI":
		return "Wi-Fi"
	case "GYM", "GYMNASIUM":
		return "Gimnasio"
	case "PARKING":
		return "Estacionamiento"
	case "ELEVATOR":
		return "Elevador"
	case "AIR_CONDITIONING", "AC":
		return "Aire Acondicionado"
	case "HEATING":
		return "Calefacción"
	case "LAUNDRY":
		return "Lavandería"
	case "PETS_ALLOWED":
		return "Mascotas Permitidas"
	case "FURNISHED":
		return "Amueblado"
	case "GARDEN":
		return "Jardín"
	case "TERRACE":
		return "Terraza"
	case "BALCONY":
		return "Balcón"
	default:
		words := strings.Split(strings.ToLower(code), "_")
		for i, w := range words {
			if len(w) > 0 {
				words[i] = strings.ToUpper(string(w[0])) + w[1:]
			}
		}

		return strings.Join(words, " ")
	}
}