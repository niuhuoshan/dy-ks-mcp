package platform

import (
	"errors"
	"testing"
)

func TestNotImplementedErrorCarriesTypedContext(t *testing.T) {
	err := NewNotImplementedError("Douyin", "Search")
	if err == nil {
		t.Fatal("NewNotImplementedError returned nil")
	}
	if !IsNotImplemented(err) {
		t.Fatalf("IsNotImplemented(%v) = false, want true", err)
	}
	if err.Error() != "douyin.Search: not implemented" {
		t.Fatalf("Error() = %q", err.Error())
	}

	var typed *NotImplementedError
	if !errors.As(err, &typed) {
		t.Fatal("errors.As did not match *NotImplementedError")
	}
	if typed.Platform != "douyin" {
		t.Fatalf("typed.Platform = %q, want douyin", typed.Platform)
	}
	if typed.Method != "Search" {
		t.Fatalf("typed.Method = %q, want Search", typed.Method)
	}
}
