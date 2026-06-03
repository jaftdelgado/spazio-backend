package visits

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		errContains string
	}{
		{
			name:        "pgError 23503 property_id",
			err:         &pgconn.PgError{Code: "23503", ConstraintName: "visits_property_id_fkey"},
			errContains: "la propiedad seleccionada no existe",
		},
		{
			name:        "pgError 23503 client_id",
			err:         &pgconn.PgError{Code: "23503", ConstraintName: "visits_client_id_fkey"},
			errContains: "el usuario involucrado no existe",
		},
		{
			name:        "pgError 23503 other",
			err:         &pgconn.PgError{Code: "23503", ConstraintName: "other"},
			errContains: "recurso relacionado no encontrado",
		},
		{
			name:        "pgError 23505",
			err:         &pgconn.PgError{Code: "23505"},
			errContains: "ya existe una visita programada",
		},
		{
			name:        "other error",
			err:         errors.New("other error"),
			errContains: "other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translated := translateError(tt.err)
			if !strings.Contains(translated.Error(), tt.errContains) {
				t.Errorf("expected %v to contain %v", translated.Error(), tt.errContains)
			}
		})
	}
}

func TestNormalizeDate(t *testing.T) {
	loc, _ := time.LoadLocation("America/Mexico_City")
	// Test with a local time
	input := time.Date(2024, 1, 1, 15, 30, 45, 123456, loc)
	expected := time.Date(2024, 1, 1, 15, 0, 0, 0, loc)
	if !expected.Equal(normalizeDate(input)) {
		t.Errorf("expected %v, got %v", expected, normalizeDate(input))
	}

	// Test with a UTC time that should be converted to local
	inputUTC := time.Date(2024, 1, 1, 15, 30, 45, 123456, time.UTC)
	normalized := normalizeDate(inputUTC)
	if normalized.Location().String() != "America/Mexico_City" {
		t.Errorf("expected location America/Mexico_City, got %v", normalized.Location())
	}
}
