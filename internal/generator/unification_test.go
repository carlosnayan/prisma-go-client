package generator

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/carlosnayan/prisma-go-client/prisma/db/utils"
)

func TestErrorUnification_ErrNotFound(t *testing.T) {
	tests := []struct {
		name     string
		inputErr error
		expected bool
	}{
		{
			name:     "SQL No Rows",
			inputErr: sql.ErrNoRows,
			expected: true,
		},
		{
			name:     "PGX No Rows",
			inputErr: errors.New("no rows in result set"),
			expected: true,
		},
		{
			name:     "Other Error",
			inputErr: errors.New("random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.MapDriverError(tt.inputErr, "First")
			if errors.Is(got, utils.ErrNotFound) != tt.expected {
				t.Errorf("expected %v, got %v for error %v", tt.expected, got, tt.inputErr)
			}
		})
	}
}
