package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLocationEnrichEvent_HasStreetAddress(t *testing.T) {
	tests := []struct {
		name        string
		event       LocationEnrichEvent
		expected    bool
		description string
	}{
		{
			name: "full address with street and house number",
			event: LocationEnrichEvent{
				PropertyID:  uuid.New(),
				Country:     "Spain",
				Street:      strPtr("Passeig de Gracia"),
				HouseNumber: strPtr("123"),
			},
			expected:    true,
			description: "Should return true when both street and house number are present",
		},
		{
			name: "only street without house number",
			event: LocationEnrichEvent{
				PropertyID: uuid.New(),
				Country:    "Spain",
				Street:     strPtr("Passeig de Gracia"),
			},
			expected:    false,
			description: "Should return false when house number is missing",
		},
		{
			name: "only house number without street",
			event: LocationEnrichEvent{
				PropertyID:  uuid.New(),
				Country:     "Spain",
				HouseNumber: strPtr("123"),
			},
			expected:    false,
			description: "Should return false when street is missing",
		},
		{
			name: "empty street and house number",
			event: LocationEnrichEvent{
				PropertyID:  uuid.New(),
				Country:     "Spain",
				Street:      strPtr(""),
				HouseNumber: strPtr(""),
			},
			expected:    false,
			description: "Should return false when both are empty strings",
		},
		{
			name: "nil street and nil house number",
			event: LocationEnrichEvent{
				PropertyID: uuid.New(),
				Country:    "Spain",
			},
			expected:    false,
			description: "Should return false when both are nil",
		},
		{
			name: "street present but empty, house number present",
			event: LocationEnrichEvent{
				PropertyID:  uuid.New(),
				Country:     "Spain",
				Street:      strPtr(""),
				HouseNumber: strPtr("123"),
			},
			expected:    false,
			description: "Should return false when street is empty string",
		},
		{
			name: "street present, house number empty",
			event: LocationEnrichEvent{
				PropertyID:  uuid.New(),
				Country:     "Spain",
				Street:      strPtr("Passeig de Gracia"),
				HouseNumber: strPtr(""),
			},
			expected:    false,
			description: "Should return false when house number is empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.HasStreetAddress()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
