package bot

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPollHTTPTimeout(t *testing.T) {
	tests := []struct {
		name        string
		listen      time.Duration
		wantTimeout time.Duration
	}{
		{name: "minimum floor", listen: 25 * time.Second, wantTimeout: 45 * time.Second},
		{name: "larger listen timeout", listen: 40 * time.Second, wantTimeout: 55 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pollHTTPTimeout(tt.listen); got != tt.wantTimeout {
				t.Fatalf("pollHTTPTimeout(%v) = %v, want %v", tt.listen, got, tt.wantTimeout)
			}
		})
	}
}

func TestSanitizeErrorRedactsToken(t *testing.T) {
	client := NewTelegramClient("123:secret", 25*time.Second)

	err := client.sanitizeError(errors.New(`Get "https://api.telegram.org/bot123:secret/getUpdates?offset=1": context deadline exceeded`))
	if err == nil {
		t.Fatal("expected sanitized error")
	}

	got := err.Error()
	if strings.Contains(got, "123:secret") {
		t.Fatalf("expected token to be redacted, got %q", got)
	}
	if !strings.Contains(got, "bot<redacted>") {
		t.Fatalf("expected redacted marker in %q", got)
	}
}
