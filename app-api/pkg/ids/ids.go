package ids

import "github.com/google/uuid"

// NewUUID creates the application-standard primary key value.
func NewUUID() (uuid.UUID, error) {
	return uuid.NewV7()
}
