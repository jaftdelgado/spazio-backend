package properties

import (
	"os"
	"strings"
	"testing"
)

func TestPropertiesSQLCoordinatesContract(t *testing.T) {
	content, err := os.ReadFile("../../../sqlc/queries/properties.sql")
	if err != nil {
		t.Fatalf("read properties.sql: %v", err)
	}

	sql := string(content)

	tests := []struct {
		name    string
		snippet string
	}{
		{
			name:    "detail reads latitude from ST_Y",
			snippet: "ST_Y(l.coordinates)::float8 AS latitude",
		},
		{
			name:    "detail reads longitude from ST_X",
			snippet: "ST_X(l.coordinates)::float8 AS longitude",
		},
		{
			name:    "create location writes point as longitude latitude",
			snippet: "ST_SetSRID(ST_MakePoint(sqlc.arg(longitude), sqlc.arg(latitude)), 4326)",
		},
		{
			name:    "update location writes point as longitude latitude",
			snippet: "coordinates = ST_SetSRID(ST_MakePoint(sqlc.arg(longitude), sqlc.arg(latitude)), 4326)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(sql, tt.snippet) {
				t.Fatalf("properties.sql missing snippet %q", tt.snippet)
			}
		})
	}

	if strings.Contains(sql, "ST_MakePoint(sqlc.arg(latitude), sqlc.arg(longitude))") {
		t.Fatal("properties.sql contains inverted ST_MakePoint(latitude, longitude)")
	}
}
