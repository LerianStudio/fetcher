# lib-streaming v3 rollout — deploy order and operator runbook

Fetcher moved from `lib-streaming/v2` to `lib-streaming/v3`. Two things in this
migration are **not** safe to discover at deploy time: the persisted outbox
envelope version, and the `ce-source` value carried on every job event. This is
the operator runbook for both.

Scope: the **Worker** only. The Manager emits no streaming events.

---

## 1. What actually changes on the wire

Almost nothing, and that is the point of reading this section before the next one.

| | Before (v2) | After (v3) |
|---|---|---|
| Destination | RabbitMQ exchange `fetcher.job.events` | unchanged |
| Routing keys | `job.completed` / `job.failed` | unchanged |
| Payload shape | snake_case, `ce-schemaversion` `2.0.0` | unchanged |
| `ce-source` | `//lerian.fetcher/worker` | **`fetcher`** |
| Persisted outbox envelope | version `1` | version `2` |

Fetcher publishes its job events to a RabbitMQ exchange, not to a Kafka topic.
The v3 "one topic per producing application" change therefore does **not** move
fetcher's traffic: every route is an explicit RabbitMQ destination, and v3 keeps
explicit destinations intact.

**No Kafka or Redpanda topic needs to be provisioned for fetcher.** The event
manifest advertises the derived names `lerian.streaming.fetcher` and
`lerian.streaming.fetcher.dlq` because the manifest contract derives them from
the ce-source, but fetcher writes neither while all of its routes are RabbitMQ.
`STREAMING_BROKERS` must still be set to something non-empty — lib-streaming's
config validation requires it when streaming is enabled — but nothing dials it:
the Kafka transport adapter is only constructed for a Kafka-kind route.

### Consumer impact: `ce-source`

The only consumer-visible change is the `ce-source` header, which goes from
`//lerian.fetcher/worker` to `fetcher`. Any consumer that filters, routes, or
audits on `ce-source` must accept the new value **before** the v3 deploy.
Consumers that bind on the exchange and routing key, or that dedupe on `ce-id`,
are unaffected — both are unchanged.

`ce-id` remains `fetcher.job.<status>.<jobID>`, and it remains the only
dedup anchor. Delivery is still at-least-once.

---

## 2. Mandatory deploy order (streaming outbox)

**Do the steps in this order. Skipping the drain destroys job events.**

Fetcher persists every job event to a durable outbox before publishing
(`DirectModeSkip` + `OutboxModeAlways`: the outbox row *is* the delivery path,
not a fallback). Both majors validate the persisted envelope's `version` field
by **strict equality**, in opposite directions: v2 accepts only version 1, v3
accepts only version 2. Neither tolerates the other's rows.

A rejected row surfaces as `ErrInvalidOutboxEnvelope`, which is classified
**non-retryable**. It is not retried and not redelivered — it is marked failed
and left there. The job event is lost, permanently.

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
correctly-addressed envelope. Two fields, both mechanical.

**Do not treat that as the plan.** It is data surgery on a durable store on the
money-adjacent path, per row, under time pressure, and it is only correct if you
have confirmed nothing else in the row is stale. The drain costs a wait. Keep the
repair knowledge for the incident where the drain was already skipped.

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
version-2 rows written by v3 are rejected by a v2 relay, non-retryably. If a
rollback is needed, apply the same procedure — quiesce intake, drain to zero on
v3, then deploy v2 and restore `STREAMING_CLOUDEVENTS_SOURCE` to its previous
value.

Consumers accepting `ce-source` `fetcher` should keep accepting it across a
rollback, so widen consumer matching rather than switching it.
