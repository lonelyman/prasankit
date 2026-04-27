package dbtypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONB map[string]any

func NewJSONB(value map[string]any) JSONB {
	if value == nil {
		return JSONB{}
	}
	return JSONB(value)
}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}

	data, err := json.Marshal(map[string]any(j))
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = JSONB{}
		return nil
	}

	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("scan JSONB: unsupported value type %T", value)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*j = JSONB(decoded)
	return nil
}
