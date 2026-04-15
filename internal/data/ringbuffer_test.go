package data

import (
	"strings"
	"testing"
)

func TestRingBuffer_BasicWrite(t *testing.T) {
	rb := newRingBuffer(16)
	rb.Write([]byte("hello"))
	if got := rb.String(); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := newRingBuffer(8)
	rb.Write([]byte("abcdefghij")) // 10 bytes > 8 cap
	got := rb.String()
	if got != "cdefghij" {
		t.Errorf("got %q, want %q", got, "cdefghij")
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	rb := newRingBuffer(8)
	rb.Write([]byte("abcde"))  // pos=5
	rb.Write([]byte("fghij")) // wraps
	got := rb.String()
	if len(got) != 8 {
		t.Errorf("len = %d, want 8", len(got))
	}
	if got != "cdefghij" {
		t.Errorf("got %q, want %q", got, "cdefghij")
	}
}

func TestRingBuffer_ManySmallWrites(t *testing.T) {
	rb := newRingBuffer(8)
	for i := 0; i < 20; i++ {
		rb.Write([]byte("x"))
	}
	got := rb.String()
	if got != strings.Repeat("x", 8) {
		t.Errorf("got %q, want %q", got, strings.Repeat("x", 8))
	}
}
