package books

import (
	"context"
	"encoding/json"
	"log/slog"
	"my-personal-lib/internal/events"

	"github.com/nats-io/nats.go/jetstream"
)

func (s *svc) publishProgressUpdated(ctx context.Context, evt events.ReadingProgressUpdated) {
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("cannot encode progress event", "error", err, "event_id", evt.EventID.String())
		return
	}

	_, err = s.js.Publish(ctx, events.SubjectProgressUpdated, data, jetstream.WithMsgID(evt.EventID.String()))
	if err != nil {
		slog.Error("cannot publiosh progress event", "erro", err, "event_id", evt.EventID.String())
		return
	}

	slog.Info("progress event published", "event_id", evt.EventID.String())
}
