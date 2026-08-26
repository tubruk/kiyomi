package queue

import (
	"errors"
	"fmt"
	"testing"
)

func TestPermanent_ClassifiesAndPreservesChain(t *testing.T) {
	base := errors.New("not found")
	wrapped := Permanent(fmt.Errorf("pull_manga: %w", base))

	if !IsPermanent(wrapped) {
		t.Fatal("expected wrapped error to be permanent")
	}
	if !errors.Is(wrapped, base) {
		t.Fatal("expected base error to remain in chain")
	}
	if wrapped.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestPermanent_NilReturnsNil(t *testing.T) {
	if Permanent(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestIsPermanent_PlainErrorFalse(t *testing.T) {
	if IsPermanent(errors.New("transient network glitch")) {
		t.Fatal("plain error should not be classified as permanent")
	}
}
