package ids_test

import (
	"testing"

	"prasankit-api/pkg/ids"
)

func TestNew_ReturnsVersion7(t *testing.T) {
	id, err := ids.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if id.Version() != 7 {
		t.Errorf("Version() = %d, want 7", id.Version())
	}
}

func TestNew_TwoCallsAreNotEqual(t *testing.T) {
	a, err := ids.New()
	if err != nil {
		t.Fatalf("New() first call error = %v", err)
	}
	b, err := ids.New()
	if err != nil {
		t.Fatalf("New() second call error = %v", err)
	}
	if a == b {
		t.Errorf("two consecutive New() calls returned equal UUIDs: %v", a)
	}
}
