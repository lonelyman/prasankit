package dbtypes

import "testing"

func TestJSONBValueDefaultsToEmptyObject(t *testing.T) {
	value, err := NewJSONB(nil).Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if value != "{}" {
		t.Fatalf("value = %v, want {}", value)
	}
}

func TestJSONBScan(t *testing.T) {
	var value JSONB
	if err := value.Scan([]byte(`{"reason":"invalid_password"}`)); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if value["reason"] != "invalid_password" {
		t.Fatalf("reason = %v, want invalid_password", value["reason"])
	}
}
