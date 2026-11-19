package events

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func mustUUID(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		t.Fatalf("UUID de teste inválido %q: %v", s, err)
	}
	return u
}

func sampleEvent(t *testing.T) ReadingProgressUpdated {
	return ReadingProgressUpdated{
		EventID:      mustUUID(t, "9c1e0a52-0000-0000-0000-000000000001"),
		Event:        EventProgressUpdated,
		BookID:       mustUUID(t, "b3f10000-0000-0000-0000-000000000002"),
		UserID:       mustUUID(t, "11111111-1111-1111-1111-111111111111"),
		PreviousPage: 100,
		CurrentPage:  120,
		TotalPages:   320,
		Status:       "reading",
		UpdatedAt:    time.Date(2026, time.October, 20, 14, 30, 0, 0, time.UTC),
	}
}

// O JSON precisa ter exatamente as chaves do contrato do CLAUDE.md:
// o books escreve com esta struct e o stats lê com ela.
func TestReadingProgressUpdatedJSONKeys(t *testing.T) {
	data, err := json.Marshal(sampleEvent(t))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal em mapa: %v", err)
	}

	want := []string{
		"book_id", "current_page", "event", "event_id", "previous_page",
		"status", "total_pages", "updated_at", "user_id",
	}
	got := make([]string, 0, len(raw))
	for k := range raw {
		got = append(got, k)
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("chaves = %v, quero %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("chaves = %v, quero %v", got, want)
		}
	}

	// Os IDs viajam como texto (não como lista de bytes) e a data em RFC 3339.
	if raw["event_id"] != "9c1e0a52-0000-0000-0000-000000000001" {
		t.Errorf("event_id = %v", raw["event_id"])
	}
	if raw["updated_at"] != "2026-10-20T14:30:00Z" {
		t.Errorf("updated_at = %v", raw["updated_at"])
	}
	if raw["event"] != "ReadingProgressUpdated" {
		t.Errorf("event = %v", raw["event"])
	}
}

// Ida e volta: o que o books publica é exatamente o que o stats vai ler.
func TestReadingProgressUpdatedRoundTrip(t *testing.T) {
	in := sampleEvent(t)

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var out ReadingProgressUpdated
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if out.EventID != in.EventID || out.BookID != in.BookID || out.UserID != in.UserID {
		t.Errorf("IDs mudaram na ida e volta: %+v", out)
	}
	if out.PreviousPage != in.PreviousPage || out.CurrentPage != in.CurrentPage || out.TotalPages != in.TotalPages {
		t.Errorf("páginas mudaram na ida e volta: %+v", out)
	}
	if out.Status != in.Status || out.Event != in.Event {
		t.Errorf("status/event mudaram na ida e volta: %+v", out)
	}
	if !out.UpdatedAt.Equal(in.UpdatedAt) {
		t.Errorf("updated_at = %v, quero %v", out.UpdatedAt, in.UpdatedAt)
	}
}
