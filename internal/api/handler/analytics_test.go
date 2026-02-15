package handler

import (
	"testing"
)

func TestAnalyticsHandler_Instantiation(t *testing.T) {
	// Verify the handler type compiles and can be instantiated.
	// Full integration tests require a database connection.
	h := NewAnalyticsHandler(nil, nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}
