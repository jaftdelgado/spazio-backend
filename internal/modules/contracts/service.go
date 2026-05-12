package contracts

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

type ContractService interface {
	GenerateContract(ctx context.Context, userID int32, input CreateContractInput) (CreateContractResult, error)
	ListContracts(ctx context.Context, userID int32, filter ListContractsFilter) ([]ContractListItem, error)
	GetContractDetail(ctx context.Context, userID int32, contractUUID uuid.UUID) (ContractDetail, error)
}

type ContractStorage interface {
	Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error
	PublicURL(ctx context.Context, storageKey string) (string, error)
}

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

func (s *service) ListContracts(ctx context.Context, userID int32, filter ListContractsFilter) ([]ContractListItem, error) {
	roleID, err := s.repository.GetUserRole(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user role: %w", err)
	}

	params := sqlcgen.ListContractsParams{
		Limit:  filter.Limit,
		Offset: (filter.Page - 1) * filter.Limit,
	}

	// 1: Admin, 2: Agent
	if roleID != 1 && roleID != 2 {
		params.OwnerID = pgtype.Int4{Int32: userID, Valid: true}
	} else if filter.OwnerID != nil {
		params.OwnerID = pgtype.Int4{Int32: *filter.OwnerID, Valid: true}
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

		clientName := ""
		if cn, ok := row.ClientName.(string); ok {
			clientName = cn
		}

		result[i] = ContractListItem{
			ContractUUID:    fmt.Sprintf("%x-%x-%x-%x-%x", row.ContractUuid.Bytes[0:4], row.ContractUuid.Bytes[4:6], row.ContractUuid.Bytes[6:8], row.ContractUuid.Bytes[8:10], row.ContractUuid.Bytes[10:16]),
			TransactionType: string(row.TransactionType),
			PropertyTitle:   row.PropertyTitle,
			AgreedAmount:    amount.Float64,
			Currency:        row.Currency,
			StartDate:       row.StartDate.Time,
			Status:          row.StatusName,
			ClientName:      clientName,
			CreatedAt:       row.CreatedAt.Time,
		}
	}

	return result, nil
}

func (s *service) GetContractDetail(ctx context.Context, userID int32, contractUUID uuid.UUID) (ContractDetail, error) {
	roleID, err := s.repository.GetUserRole(ctx, userID)
	if err != nil {
		return ContractDetail{}, fmt.Errorf("get user role: %w", err)
	}

	row, err := s.repository.GetContractByUUID(ctx, contractUUID)
	if err != nil {
		return ContractDetail{}, fmt.Errorf("get contract by uuid: %w", err)
	}

	if roleID != 1 && roleID != 2 && row.OwnerID != userID {
		return ContractDetail{}, fmt.Errorf("no tiene permiso para ver este contrato")
	}

	amount, _ := row.AgreedAmount.Float64Value()

	pdfURL, _ := s.storage.PublicURL(ctx, row.StorageKey)

	var endDate *time.Time
	if row.EndDate.Valid {
		endDate = &row.EndDate.Time
	}

	return ContractDetail{
		ContractUUID:  fmt.Sprintf("%x-%x-%x-%x-%x", row.ContractUuid.Bytes[0:4], row.ContractUuid.Bytes[4:6], row.ContractUuid.Bytes[6:8], row.ContractUuid.Bytes[8:10], row.ContractUuid.Bytes[10:16]),
		PropertyTitle: row.PropertyTitle,
		OwnerName:     row.OwnerFirstName + " " + row.OwnerLastName,
		ClientName:    row.ClientFirstName + " " + row.ClientLastName,
		AgreedAmount:  amount.Float64,
		Currency:      row.Currency,
		StartDate:     row.StartDate.Time,
		EndDate:       endDate,
		Status:        row.StatusName,
		PDFUrl:        pdfURL,
	}, nil
}

func (s *service) GenerateContract(ctx context.Context, userID int32, input CreateContractInput) (CreateContractResult, error) {
	if input.EndDate != nil && !input.EndDate.After(input.StartDate) {
		return CreateContractResult{}, fmt.Errorf("la fecha de finalización debe ser posterior a la fecha de inicio")
	}

	data, err := s.repository.GetContractDataByTransactionID(ctx, input.TransactionID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("fetch transaction data: %w", err)
	}

	if data.OwnerID != userID {
		return CreateContractResult{}, fmt.Errorf("solo el propietario de la propiedad puede generar el contrato")
	}

	clauses, err := s.repository.GetPropertyClausesByTransactionID(ctx, input.TransactionID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("fetch property clauses: %w", err)
	}

	contractUUID := uuid.New()

	pdfBytes, err := s.generatePDF(data, clauses, input, contractUUID)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("generate pdf: %w", err)
	}

	storageKey := fmt.Sprintf("contracts/%s.pdf", contractUUID.String())
	err = s.storage.Upload(ctx, storageKey, "application/pdf", bytes.NewReader(pdfBytes))
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("upload pdf: %w", err)
	}

	_, err = s.repository.CreateContract(ctx, input, storageKey)
	if err != nil {
		return CreateContractResult{}, fmt.Errorf("create contract record: %w", err)
	}

	return CreateContractResult{
		ContractUUID: contractUUID.String(),
		StorageKey:   storageKey,
	}, nil
}

func (s *service) generatePDF(data sqlcgen.GetContractDataByTransactionIDRow, clauses []sqlcgen.GetPropertyClausesByTransactionIDRow, input CreateContractInput, contractUUID uuid.UUID) ([]byte, error) {
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
	if data.TransactionType == sqlcgen.TransactionTypeSale {
		title = "CONTRATO DE COMPRAVENTA"
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		Build()

	m := maroto.New(cfg)

	m.AddRows(
		row.New(20).Add(
			col.New(12).Add(
				text.New(title, props.Text{
					Top:   5,
					Size:  16,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		),
		row.New(10).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Folio: %s", contractUUID.String()), props.Text{
					Size:  10,
					Align: align.Right,
				}),
			),
		),
		row.New(10).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Fecha de Emisión: %s", time.Now().Format("02 de January de 2006")), props.Text{
					Size:  10,
					Align: align.Right,
				}),
			),
		),
		line.NewRow(5),
	)

	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New("1. PARTES CONTRATANTES", props.Text{Style: fontstyle.Bold, Size: 12}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("EL PROPIETARIO: %s %s", data.OwnerFirstName, data.OwnerLastName), props.Text{Size: 10}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Correo: %s", data.OwnerEmail), props.Text{Size: 10}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("EL CLIENTE: %s %s", data.ClientFirstName, data.ClientLastName), props.Text{Size: 10}),
			),
		),
		row.New(15).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Correo: %s", data.ClientEmail), props.Text{Size: 10}),
			),
		),
	)

	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New("2. OBJETO DEL CONTRATO", props.Text{Style: fontstyle.Bold, Size: 12}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("La propiedad denominada '%s', ubicada en:", data.PropertyTitle), props.Text{Size: 10}),
			),
		),
		row.New(15).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("Calle %s No. %s, Col. %s, %s, %s.", data.Street, data.ExteriorNumber, data.Neighborhood, data.CityName, data.StateName), props.Text{Size: 10}),
			),
		),
	)

	m.AddRows(
		row.New(10).Add(
			col.New(12).Add(
				text.New("3. TÉRMINOS FINANCIEROS", props.Text{Style: fontstyle.Bold, Size: 12}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("MONTO ACORDADO: %.2f %s", input.AgreedAmount, input.Currency), props.Text{Size: 10}),
			),
		),
		row.New(8).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("FECHA DE INICIO: %s", input.StartDate.Format("02/01/2006")), props.Text{Size: 10}),
			),
		),
	)

	if input.EndDate != nil {
		m.AddRows(
			row.New(8).Add(
				col.New(12).Add(
					text.New(fmt.Sprintf("FECHA DE VENCIMIENTO: %s", input.EndDate.Format("02/01/2006")), props.Text{Size: 10}),
				),
			),
		)
	}

	m.AddRows(
		row.New(15).Add(
			col.New(12).Add(
				text.New(fmt.Sprintf("DURACIÓN TOTAL: %s", duration), props.Text{Size: 10}),
			),
		),
	)

	if len(clauses) > 0 {
		m.AddRows(
			row.New(10).Add(
				col.New(12).Add(
					text.New("4. CLÁUSULAS ADICIONALES", props.Text{Style: fontstyle.Bold, Size: 12}),
				),
			),
		)

		for i, c := range clauses {
			val := ""
			switch c.ValueTypeCode {
			case "BOOLEAN":
				if c.BooleanValue.Bool {
					val = "Sí"
				} else {
					val = "No"
				}
			case "INTEGER":
				val = fmt.Sprintf("%d", c.IntegerValue.Int32)
			case "RANGE":
				val = fmt.Sprintf("Desde %v hasta %v", c.MinValue, c.MaxValue)
			}

			m.AddRows(
				row.New(8).Add(
					col.New(12).Add(
						text.New(fmt.Sprintf("%d. %s: %s", i+1, c.ClauseName, val), props.Text{Size: 10}),
					),
				),
			)
			description := ""
			if c.ClauseDescription.Valid {
				description = strings.TrimSpace(c.ClauseDescription.String)
			}
			if description != "" {
				m.AddRows(
					row.New(8).Add(
						col.New(12).Add(
							text.New(fmt.Sprintf("   (%s)", description), props.Text{Size: 9, Style: fontstyle.Italic}),
						),
					),
				)
			}
		}
	}

	m.AddRows(
		row.New(40),
		row.New(20).Add(
			col.New(5).Add(
				line.New(props.Line{Thickness: 0.5}),
			),
			col.New(2),
			col.New(5).Add(
				line.New(props.Line{Thickness: 0.5}),
			),
		),
		row.New(10).Add(
			col.New(5).Add(
				text.New("EL PROPIETARIO", props.Text{Align: align.Center, Size: 10}),
			),
			col.New(2),
			col.New(5).Add(
				text.New("EL CLIENTE", props.Text{Align: align.Center, Size: 10}),
			),
		),
		row.New(10).Add(
			col.New(5).Add(
				text.New(fmt.Sprintf("%s %s", data.OwnerFirstName, data.OwnerLastName), props.Text{Align: align.Center, Size: 10}),
			),
			col.New(2),
			col.New(5).Add(
				text.New(fmt.Sprintf("%s %s", data.ClientFirstName, data.ClientLastName), props.Text{Align: align.Center, Size: 10}),
			),
		),
	)

	document, err := m.Generate()
	if err != nil {
		return nil, err
	}

	return document.GetBytes(), nil
}
