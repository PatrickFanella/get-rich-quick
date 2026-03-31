package cli

import "testing"

func TestTruncatePreservesUTF8Runes(t *testing.T) {
	t.Parallel()

	got := truncateString("héllo世界", 6)
	want := "héllo…"
	if got != want {
		t.Fatalf("truncateString() = %q, want %q", got, want)
	}
}
