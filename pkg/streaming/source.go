// Package streaming holds fetcher's boot-time guards for the lib-streaming
// contract.
package streaming

import (
	"errors"
	"fmt"
)

// ErrSourceNotRoster is returned when the configured CloudEvents source is not
// the application's roster name. Exported so bootstrap tests can assert the
// fail-closed contract with errors.Is rather than by matching message text.
var ErrSourceNotRoster = errors.New("streaming: configured ce-source is not this application's roster name")

// RequireRosterSource fails closed unless the configured ce-source is exactly
// the application's roster name.
//
// A grammar-legal source is not a usable one. lib-streaming accepts any single
// dot-free lowercase segment, so "fetcher-worker" or "fetcher_v2" passes its
// validation while being wrong: the ce-source IS the application's identity on
// every event it emits, and it is the sole input to the topic and DLQ names the
// platform provisions. Those grants are literal patterns on the ROSTER name,
// never prefixes — a prefixed grant on "lerian.streaming.fetcher" would also
// authorize "lerian.streaming.fetcherx", reaching a neighbouring application's
// stream.
//
// Fetcher's own job events currently ride a RabbitMQ exchange rather than a
// Kafka topic, so a wrong source here corrupts event identity rather than
// destroying delivery: every consumer keying off ce-source stops matching, and
// the DLQ a failed publish falls back to carries a name nothing owns. The day
// fetcher gains a Kafka route, the same misconfiguration becomes total silent
// event loss instead. Equality against the roster name is what forecloses both,
// which is why this is an equality test and not a shape test.
//
// This is checked whether or not streaming is ENABLED. lib-streaming skips its
// own source validation entirely for a disabled config, so a source left over
// from the pre-v3 URI shape would otherwise sit in an env file until someone
// flipped the flag.
func RequireRosterSource(configured, roster string) error {
	if configured != roster {
		return fmt.Errorf(
			"%w: STREAMING_CLOUDEVENTS_SOURCE must be %q, got %q — the broker topics and ACLs are provisioned for the roster name only",
			ErrSourceNotRoster, roster, configured,
		)
	}

	return nil
}
