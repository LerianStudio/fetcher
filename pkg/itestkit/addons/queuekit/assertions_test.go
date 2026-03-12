package queuekit

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

type queuePayload struct {
	ID    string `json:"id"`
	Group string `json:"group"`
	Step  int    `json:"step"`
}

func sampleParsedMessage() ParsedMessage[queuePayload] {
	return ParsedMessage[queuePayload]{
		Message: Message{
			Body:          []byte(`{"id":"job-1","nested":{"step":2}}`),
			Headers:       map[string]any{"x-attempt": 2, "x-trace": "trace-1"},
			RoutingKey:    "jobs.created",
			MessageID:     "msg-1",
			CorrelationID: "corr-1",
			ContentType:   "application/json",
			Timestamp:     time.Unix(1700000000, 0),
		},
		Payload: queuePayload{ID: "job-1", Group: "primary", Step: 2},
	}
}

func TestQueueAssertionsHelpers(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "message assertions chain on matching metadata and payload",
			run: func(t *testing.T) {
				t.Parallel()

				msg := sampleParsedMessage()
				assertions := AssertMessage(t, msg).
					HasRoutingKey("jobs.created").
					HasHeader("x-attempt", 2).
					HasHeaderKey("x-trace").
					HasCorrelationID("corr-1").
					HasMessageID("msg-1").
					HasContentType("application/json").
					PayloadEquals(queuePayload{ID: "job-1", Group: "primary", Step: 2}).
					PayloadSatisfies("step is positive", func(p queuePayload) bool { return p.Step > 0 })

				if assertions.Payload() != msg.Payload {
					t.Fatalf("expected payload getter to return original payload")
				}

				if !reflect.DeepEqual(assertions.Message(), msg) {
					t.Fatalf("expected message getter to return original parsed message")
				}
			},
		},
		{
			name: "result assertions expose counts and indexed messages",
			run: func(t *testing.T) {
				t.Parallel()

				result := WaitResult[queuePayload]{
					Messages: []ParsedMessage[queuePayload]{
						sampleParsedMessage(),
						{Payload: queuePayload{ID: "job-2", Group: "secondary", Step: 3}},
					},
					Unmatched: []Message{{RoutingKey: "jobs.ignored"}},
					Duration:  25 * time.Millisecond,
				}

				visited := make([]string, 0, 2)
				assertions := AssertResult(t, result).
					HasCount(2).
					HasAtLeast(1).
					HasNoErrors().
					DidNotTimeout().
					UnmatchedCount(1).
					All(func(_ *testing.T, _ int, msg ParsedMessage[queuePayload]) {
						visited = append(visited, msg.Payload.ID)
					})

				if got := assertions.First().Payload().ID; got != "job-1" {
					t.Fatalf("expected first message payload, got %q", got)
				}

				if got := assertions.At(1).Payload().ID; got != "job-2" {
					t.Fatalf("expected indexed message payload, got %q", got)
				}

				if !reflect.DeepEqual(visited, []string{"job-1", "job-2"}) {
					t.Fatalf("unexpected visited payloads: %#v", visited)
				}

				if !reflect.DeepEqual(assertions.Messages(), result.Messages) {
					t.Fatalf("expected Messages accessor to return result messages")
				}

				if !reflect.DeepEqual(assertions.Result(), result) {
					t.Fatalf("expected Result accessor to return original result")
				}
			},
		},
		{
			name: "result assertions fail fast for missing indexes",
			run: func(t *testing.T) {
				t.Parallel()

				result := WaitResult[queuePayload]{Messages: []ParsedMessage[queuePayload]{sampleParsedMessage()}}
				if got := AssertResult(t, result).At(0).Payload().ID; got != "job-1" {
					t.Fatalf("expected At(0) to return first message, got %q", got)
				}
			},
		},
		{
			name: "json helpers compare normalized payloads and surface failures",
			run: func(t *testing.T) {
				t.Parallel()

				expected := []byte(`{"items":[{"id":1}],"meta":{"count":1}}`)
				actual := []byte("{\n  \"meta\": {\"count\": 1}, \"items\": [{\"id\": 1}]\n}")
				if !JSONEqual(t, expected, actual) {
					t.Fatalf("expected JSONEqual to ignore formatting differences")
				}

				AssertJSONEqual(t, expected, actual)
				AssertJSONField(t, actual, "meta.count", 1)
			},
		},
		{
			name: "message sequences preserve ordering grouping and filtering semantics",
			run: func(t *testing.T) {
				t.Parallel()

				messages := []ParsedMessage[queuePayload]{
					sampleParsedMessage(),
					{Message: Message{RoutingKey: "jobs.updated"}, Payload: queuePayload{ID: "job-2", Group: "secondary", Step: 2}},
					{Message: Message{RoutingKey: "jobs.updated"}, Payload: queuePayload{ID: "job-3", Group: "primary", Step: 3}},
				}

				sequence := NewSequence(messages)
				if !reflect.DeepEqual(sequence.RoutingKeysInOrder(), []string{"jobs.created", "jobs.updated", "jobs.updated"}) {
					t.Fatalf("unexpected routing key order: %#v", sequence.RoutingKeysInOrder())
				}

				sequence.AssertOrder(t, func(p queuePayload) string { return p.ID }, []string{"job-1", "job-2", "job-3"})

				filtered := sequence.FilterBy(func(p queuePayload) bool { return p.Group == "primary" })
				if len(filtered) != 2 {
					t.Fatalf("expected filtered sequence length 2, got %d", len(filtered))
				}

				groups := sequence.GroupBy(func(p queuePayload) string { return p.Group })
				if len(groups["primary"]) != 2 || len(groups["secondary"]) != 1 {
					t.Fatalf("unexpected grouped sequence: %#v", groups)
				}
			},
		},
		{
			name: "expect helper tracks failures and summarizes results",
			run: func(t *testing.T) {
				t.Parallel()

				result := WaitResult[queuePayload]{
					Messages: []ParsedMessage[queuePayload]{sampleParsedMessage()},
					Errors:   []error{},
					TimedOut: false,
					Duration: 42 * time.Millisecond,
				}

				helper := ExpectMessages(t, result).
					ToSucceed().
					ToHaveCount(1).
					ToContainWhere("job-1 exists", func(p queuePayload) bool { return p.ID == "job-1" })

				if helper.Failed() {
					t.Fatalf("expected helper to remain successful")
				}

				helper.OrFatal()

				summary := Summary(result)
				for _, fragment := range []string{"matched=1", "errors=0", "timedOut=false"} {
					if !strings.Contains(summary, fragment) {
						t.Fatalf("expected summary to contain %q, got %q", fragment, summary)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
