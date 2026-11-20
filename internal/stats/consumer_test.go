package stats

import (
	"errors"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
)

// fakeMsg imita uma mensagem do JetStream sem NATS rodando.
// A interface jetstream.Msg fica embutida (nil) só para o tipo satisfazê-la;
// os métodos que o handler usa são sobrescritos abaixo e registram o que aconteceu.
type fakeMsg struct {
	jetstream.Msg

	data []byte

	acked, naked, termed bool
}

func (m *fakeMsg) Data() []byte    { return m.data }
func (m *fakeMsg) Subject() string { return "books.progress.updated" }
func (m *fakeMsg) Ack() error      { m.acked = true; return nil }
func (m *fakeMsg) Nak() error      { m.naked = true; return nil }
func (m *fakeMsg) Term() error     { m.termed = true; return nil }

const validEvent = `{
	"event_id": "9c1e0a52-0000-0000-0000-000000000001",
	"event": "ReadingProgressUpdated",
	"book_id": "b3f10000-0000-0000-0000-000000000002",
	"user_id": "11111111-1111-1111-1111-111111111111",
	"previous_page": 100,
	"current_page": 120,
	"total_pages": 320,
	"status": "reading",
	"updated_at": "2026-10-20T14:30:00Z"
}`

func TestHandleProgressMsg(t *testing.T) {
	tests := []struct {
		name string

		data       string
		processErr error // o que o service falso devolve

		wantProcessCalled bool
		wantAck           bool
		wantNak           bool
		wantTerm          bool
	}{
		{
			name:              "evento válido é gravado e confirmado (Ack)",
			data:              validEvent,
			wantProcessCalled: true,
			wantAck:           true,
		},
		{
			name:              "falha ao gravar devolve a mensagem para reentrega (Nak)",
			data:              validEvent,
			processErr:        errors.New("connection refused"),
			wantProcessCalled: true,
			wantNak:           true,
		},
		{
			name:     "JSON quebrado é descartado para sempre (Term)",
			data:     `{"event_id": `,
			wantTerm: true,
		},
		{
			name:     "JSON válido sem event_id é descartado (Term)",
			data:     `{"book_id": "b3f10000-0000-0000-0000-000000000002", "current_page": 10}`,
			wantTerm: true,
		},
		{
			name:     "objeto vazio é descartado (Term)",
			data:     `{}`,
			wantTerm: true,
		},
		{
			name:     "event_id que não é UUID é descartado (Term)",
			data:     `{"event_id": "abc"}`,
			wantTerm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{processErr: tt.processErr}
			msg := &fakeMsg{data: []byte(tt.data)}

			handleProgressMsg(svc, msg)

			if svc.processCalled != tt.wantProcessCalled {
				t.Errorf("service chamado = %v, quero %v", svc.processCalled, tt.wantProcessCalled)
			}
			if msg.acked != tt.wantAck || msg.naked != tt.wantNak || msg.termed != tt.wantTerm {
				t.Errorf("ack/nak/term = %v/%v/%v, quero %v/%v/%v",
					msg.acked, msg.naked, msg.termed, tt.wantAck, tt.wantNak, tt.wantTerm)
			}

			// No caso válido, o service recebe o evento decodificado por completo.
			if tt.wantProcessCalled {
				if got := svc.gotEvent.EventID.String(); got != "9c1e0a52-0000-0000-0000-000000000001" {
					t.Errorf("event_id repassado = %s", got)
				}
				if svc.gotEvent.PreviousPage != 100 || svc.gotEvent.CurrentPage != 120 {
					t.Errorf("páginas repassadas = %d → %d, quero 100 → 120",
						svc.gotEvent.PreviousPage, svc.gotEvent.CurrentPage)
				}
			}
		})
	}
}
