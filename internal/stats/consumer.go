package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"my-personal-lib/internal/events"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func handleProgressMsg(service StatsServices, msg jetstream.Msg) {
	var evt events.ReadingProgressUpdated
	err := json.Unmarshal(msg.Data(), &evt)
	if err != nil {
		slog.Error("invalid event discarded", "error", err, "subject", msg.Subject())
		msg.Term()
		return
	}

	if !evt.EventID.Valid {
		slog.Error("event without event_id discarded", "subject", msg.Subject())
		msg.Term()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = service.ProcessProgressUpdated(ctx, evt)
	if err != nil {
		slog.Error("cannot process event", "error", err, "event_id", evt.EventID.String())
		msg.Nak()
		return
	}

	msg.Ack()
}

func StartConsumer(ctx context.Context, js jetstream.JetStream, service StatsServices) (jetstream.ConsumeContext, error) {
	cons, err := js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
		Durable:       "stats",
		FilterSubject: events.SubjectProgressUpdated,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("create consumer: %w", err)
	}
	cc, err := cons.Consume(func(msg jetstream.Msg) { handleProgressMsg(service, msg) })
	if err != nil {
		return nil, fmt.Errorf("start consuming: %w", err)
	}

	return cc, nil
}
