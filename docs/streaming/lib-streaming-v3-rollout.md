# lib-streaming v3 rollout — deploy order and operator runbook

Fetcher moved from `lib-streaming/v2` to `lib-streaming/v3`. Two things in this
migration are **not** safe to discover at deploy time: the persisted outbox
envelope version, and the `ce-source` value carried on every job event. This is
the operator runbook for both.

Scope: the **Worker** only. The Manager emits no streaming events.

---

## 1. What actually changes on the wire

Where the events go does not change. Two CloudEvents headers do.

| | Before (v2) | After (v3) |
|---|---|---|
| Destination | RabbitMQ exchange `fetcher.job.events` | unchanged |
| Routing keys | `job.completed` / `job.failed` | unchanged |
| Payload shape | snake_case, `ce-schemaversion` `2.0.0` | unchanged |
| `ce-id` | `fetcher.job.<status>.<jobID>` | unchanged |
| `ce-source` | `//lerian.fetcher/worker` | **`fetcher`** |
| `ce-type` (completed) | `studio.lerian.job.completed` | **`studio.lerian.fetcher.job.completed`** |
| `ce-type` (failed) | `studio.lerian.job.failed` | **`studio.lerian.fetcher.job.failed`** |
| Persisted outbox envelope | version `1` | version `2` |

Fetcher publishes its job events to a RabbitMQ exchange, not to a Kafka topic.
The v3 "one topic per producing application" change therefore does **not** move
fetcher's traffic: every route is an explicit RabbitMQ destination, and v3 keeps
explicit destinations intact.

**No Kafka or Redpanda topic needs to be provisioned for fetcher, and fetcher
publishes no event manifest.** The lib-streaming manifest handler is deliberately
not mounted — serving event topology from the Worker's probe port would leak it,
and the 404 is pinned by test. The derived names `lerian.streaming.fetcher` and
`lerian.streaming.fetcher.dlq` exist only as an assertion in fetcher's own
contract test, to catch the library changing how it derives names from the
ce-source. Nothing writes them, nothing reads them, and no Lerian tooling
provisions topics from manifests in any case.

`STREAMING_BROKERS` must still be set to something non-empty — lib-streaming's
config validation requires it when streaming is enabled — but nothing dials it:
the Kafka transport adapter is only constructed for a Kafka-kind route.

### Consumer impact: `ce-source` and `ce-type`

Both changed headers reach the AMQP message headers verbatim, so this is a real
consumer-visible change, not an internal one.

`ce-source` goes from `//lerian.fetcher/worker` to `fetcher`. `ce-type` gains the
source as a new segment — v3 composes it as
`studio.lerian.<source>.<resourceType>.<eventType>` rather than
`studio.lerian.<resourceType>.<eventType>` — so both event types are renamed on
the wire. The segment was added upstream because without it two services
publishing the same resource and event names produce byte-identical ce-type
values.

**Any consumer that filters, routes, or audits on `ce-source` or `ce-type` must
accept the new values before the v3 deploy.** Widen the match rather than
switching it, so the same consumer survives a rollback.

Consumers that bind on the exchange and routing key, or that dedupe on `ce-id`,
are unaffected — both are unchanged. `ce-id` remains
`fetcher.job.<status>.<jobID>` and remains the only dedup anchor. Delivery is
still at-least-once.

Fetcher pins both `ce-type` values in `TestJobEventStreamingContract_CeTypeValues`
so a future library bump breaks the build rather than a subscription. Nothing
pinned any `ce-*` header before v3, which is how this change nearly shipped
undocumented.

---

## 2. Mandatory deploy order (streaming outbox)

**Do the steps in this order. Skipping the drain destroys job events.**

Fetcher persists every job event to a durable outbox before publishing
(`DirectModeSkip` + `OutboxModeAlways`: the outbox row *is* the delivery path,
not a fallback). Both majors validate the persisted envelope's `version` field
by **strict equality**, in opposite directions: v2 accepts only version 1, v3
accepts only version 2. Neither tolerates the other's rows.

A rejected row surfaces as `ErrInvalidOutboxEnvelope`. Fetcher wires **no retry
classifier** on the outbox dispatcher, so that error is treated as **retryable**:
the row is retried on every dispatch cycle until it exhausts the dispatch-attempt
ceiling (10 by default), and only then is it marked failed. It will never
succeed — the version check is deterministic — so the retries are certain to be
wasted, but they are not instantaneous.

That is a feature here, not an oversight. **It buys a repair window.** Between the
deploy and the ceiling, an affected row is still sitting in the collection as
`PENDING` or `FAILED` with a climbing `attempts` count, holding its full payload.
That is precisely the window in which the two-field repair described in §3 is
viable. Leaving the error retryable is deliberate: for mandatory job events, a
narrower window is the wrong direction — classifying it non-retryable would
convert a recoverable state into an immediate, silent write-off.

The window is not generous, and it is not a substitute for the drain. Once the
ceiling is reached the row is failed and the job event is lost.

### (a) Quiesce job intake on the **v2** build, keeping the Worker running

Stop submitting new extraction jobs and let in-flight jobs finish. The Worker
must stay up: its outbox relay is what drains the rows.

> **Fetcher cannot disable streaming during the rollout window.** Other services
> set `STREAMING_ENABLED=false` for the switchover so no build writes fresh rows.
> Fetcher's Worker *refuses to start* unless `STREAMING_ENABLED=true`, because
> `job.completed` / `job.failed` are a mandatory product contract with no
> silent-degradation path. Quiescing intake is the equivalent lever: no new
> terminal job, no new outbox row.

### (b) Drain the outbox to zero, with the relay live

Draining means "let v2 ship the rows it can still address", not "delete the rows".

Run against **every** database that holds a `streaming_outbox_events`
collection (see *Where to run this*, below):

```javascript
db.streaming_outbox_events.countDocuments({
  event_type: "lerian.streaming.publish",
  status: { $ne: "PUBLISHED" }
})
```

Wait until this returns `0` everywhere. `PUBLISHED` is the only terminal-success
state; `PENDING`, `PROCESSING`, `FAILED` and `INVALID` are all rows that have not
shipped yet.

> **`INVALID` rows never drain.** The predicate above counts them, and the relay
> will not retry them — `INVALID` is a terminal state. A deployment that already
> has `INVALID` rows can therefore never reach zero by waiting. Those rows need
> manual disposition before the deploy: establish why each one was rejected, and
> either repair or consciously write it off. Do not work around this by narrowing
> the predicate to exclude `INVALID` — that hides exactly the rows most likely to
> represent a lost job event.

### (c) Set the new ce-source and roll out v3

1. Set `STREAMING_CLOUDEVENTS_SOURCE=fetcher`. This is **required**: the Worker
   validates it at startup against fetcher's roster name and refuses to boot on
   any other value — including the old `//lerian.fetcher/worker`, which v3's own
   grammar also rejects. The check runs whether or not streaming is enabled, so a
   stale value cannot lie dormant in an env file.
2. Deploy the v3 build.
3. Resume job intake.

No exchange or binding changes are needed; `RABBITMQ_JOB_EVENTS_EXCHANGE` keeps
its value.

### Where to run the drain check

**Multi-tenant deployments: every tenant database, not just the shared one.**
Fetcher's streaming outbox is tenant-scoped when `MULTI_TENANT_ENABLED=true` —
each tenant's rows live in that tenant's own database, and the dispatcher routes
per tenant. A clean shared database proves nothing about the others. Enumerate
tenants the way the Worker does: the active tenants for service `fetcher` from
the Tenant Manager.

**Single-tenant deployments:** one database, the Worker's configured Mongo
database. Rows carry the stable tenant id `single-tenant`.

---

## 3. Why the drain cannot be skipped, and why the repairer does not cover it

Fetcher has a terminal-event repairer that re-emits when a job's
`terminalEventPending` metadata flag survives a crash. It is tempting to assume
it would rescue any row v3 rejects. **It would not**, and the ordering is why:

1. the job is marked terminal with `terminalEventPending: true`,
2. the event is emitted — which **writes the outbox row**,
3. the flag is cleared.

The flag is cleared once the row is *durably written*, not once the broker has
the event. So a version-1 row still sitting in the outbox, whose job flag has
already been cleared, has **no repair path**: the repairer will not re-emit it,
because as far as the job record is concerned the event was handed off.

Those rows are exactly what the drain protects. Jobs whose flag is *still set*
are safe either way — v3 re-emits them with a fresh version-2 envelope.

### Fetcher's wrinkle: a rejected row would have delivered fine

Worth stating plainly, because it changes how the failure reads in a
post-mortem. In services whose destination changed between majors, a stale
envelope also points at a dead topic, so its rejection is arguably protective.
**Fetcher's persisted destination is byte-identical across the two versions** —
the same exchange, the same routing key — and the envelope and event structs are
unchanged too. A version-1 row would have published correctly in every respect.
It is refused *purely* on the integer in its `version` field.

That makes a manual repair genuinely conceivable for fetcher, unlike elsewhere:
rewriting `version` to `2` and `event.source` to `fetcher` would produce a valid,
correctly-addressed envelope. Two fields, both mechanical. `ce-source` and
`ce-type` are both derived from `event.source` at publish time, so fixing that one
field corrects both headers — there is no third edit.

And because `ErrInvalidOutboxEnvelope` is retryable here (see §2), there is an
actual window to do it in: an affected row stays in the collection with its
payload intact until the dispatch-attempt ceiling is reached. Repair it before
then and the relay ships it on the next cycle with no further intervention.

**Do not treat that as the plan.** It is data surgery on a durable store on the
money-adjacent path, per row, against a closing window, and it is only correct if
you have confirmed nothing else in the row is stale. The drain costs a wait. Keep
the repair knowledge for the incident where the drain was already skipped.

### If the drain does not reach zero

**Do not deploy.** Inspect what is stuck:

```javascript
db.streaming_outbox_events.find(
  { event_type: "lerian.streaming.publish", status: { $ne: "PUBLISHED" } },
  { id: 1, status: 1, attempts: 1, last_error: 1, created_at: 1, payload: 1 }
).sort({ created_at: 1 })
```

`payload` is stored as a JSON string, so read the `version` and
`definition_key` out of it directly rather than querying them as fields.

Resolve why the relay cannot ship — broker reachability, a tenant database the
dispatcher cannot resolve, rows already `FAILED` past their attempt ceiling —
before rolling forward.

### This is likely a non-event, but confirm it

Fetcher requires `STREAMING_ENABLED=true` to boot, so unlike services that ship
with streaming off, fetcher environments have been writing outbox rows all along.
The drain is therefore a real step here, not a formality.

What makes it usually quick is that the relay ships continuously: a healthy
Worker's outbox is near-empty at any moment. Expect the count to reach zero
within one relay interval of intake stopping. If it does not, that is a
pre-existing delivery problem the deploy would have converted into data loss.

---

## 4. Rollback

Rolling back from v3 to v2 has the same hazard in the opposite direction:
version-2 rows written by v3 are rejected by a v2 relay, with the same retryable
classification and the same finite window before the attempt ceiling writes them
off. If a rollback is needed, apply the same procedure — quiesce intake, drain to
zero on v3, then deploy v2 and restore `STREAMING_CLOUDEVENTS_SOURCE` to its
previous value.

Consumers should keep accepting the v3 `ce-source` and `ce-type` values across a
rollback rather than switching back, so the same consumer survives both
directions without a coordinated redeploy.

---

## 5. Residual risks and known gaps

**Version skew during a rolling deploy is near-silent.** While Manager and Worker
pods run different builds, a Worker can receive a job message whose AMQP envelope
its build does not agree with. That takes the same path as a cross-tenant replay:
the message is rejected, acknowledged, and the only trace is one log line. There
is no metric and no DLQ, so a skew window that drops work looks identical to a
quiet period on every dashboard.

Two consequences for the deploy:

- Keep the skew window short, and prefer quiescing intake (which you are doing
  anyway for the outbox drain) over relying on both builds interoperating.
- If job counts look wrong afterward, the evidence is in Worker logs, not in
  metrics. Search for the tenant-header mismatch and signature-verification
  messages.

Closing this gap means a counter on envelope-rejection paths, keyed by reason.
That is a follow-up, not part of this migration — but it is the reason a skewed
rollout is worth avoiding rather than measuring.
