# Diagramas de sequência — `/readyz`

Cinco diagramas Mermaid cobrindo os fluxos principais de funcionamento. Leia em ordem — cada um se apoia no contexto do anterior.

> Os diagramas usam apenas atores de alto nível para ficarem legíveis. Detalhes de chamadas internas (locks, atomic loads, struct field reads) ficam de fora.

---

## Como ler

Os atores que aparecem repetidamente:

| Ator | Onde mora | Papel |
|---|---|---|
| `K8s` | kubelet / kube-proxy | Faz probes e roteia tráfego. |
| `Bootstrap` | `components/{manager,worker}/internal/bootstrap/config.go` | Orquestra inicialização. |
| `Server` | `bootstrap/server.go` (manager) ou `health_server.go` (worker) | Servidor HTTP Fiber. |
| `Routes` | `routes.go` (manager) ou montagem direta no health server (worker) | Roteador Fiber. |
| `Handler` | `pkg/bootstrap/readyz/handler.go` | Motor do `/readyz`. |
| `Checker` | `pkg/bootstrap/readyz/checker_*.go` | Probers individuais (Mongo, Redis, etc). |
| `Dep` | externa | MongoDB, RabbitMQ, Redis, S3, Tenant Manager. |
| `State` | `pkg/bootstrap/readyz/state.go` | Flags atomic (`drainingState`, `selfProbeOK`). |
| `Metrics` | `pkg/bootstrap/readyz/metrics.go` | Coletores Prometheus. |

---

## Diagrama 1 — Bootstrap (boot do pod)

Mostra a sequência completa do startup: do momento em que o `main` é chamado até o servidor estar aceitando requests. As caixas em multi-cor representam os pontos onde diferentes arquivos contribuem.

```mermaid
sequenceDiagram
    autonumber
    participant Main as main()
    participant Boot as Bootstrap<br/>(config.go)
    participant TLS as ValidateSaaSTLS<br/>(tls_enforcement.go)
    participant Init as init* funcs<br/>(Mongo, RMQ, Redis...)
    participant Adapter as readyz_adapters.go<br/>(buildCheckers)
    participant SelfProbe as RunSelfProbe<br/>(selfprobe.go)
    participant State as State<br/>(state.go)
    participant Server as Server / HealthServer
    participant K8s as K8s readinessProbe

    Main->>Boot: InitServers / InitWorker
    Boot->>Boot: load Config (env vars)

    Note over Boot,TLS: Gate de TLS — antes de qualquer conexão
    Boot->>TLS: validateSaaSTLSFn(saasCfg)
    alt DEPLOYMENT_MODE = saas e dep sem TLS
        TLS-->>Boot: ErrSaaSTLSRequired
        Boot-->>Main: erro — boot abortado
    else OK
        TLS-->>Boot: nil
    end

    Note over Boot,Init: Só agora abre conexões reais
    Boot->>Init: initMongoRepositories / initStorageRepository / ...
    Init-->>Boot: clientes prontos (mongoClient, s3Client, rabbitAdapter, ...)

    Boot->>Adapter: buildManagerReadyzCheckers / buildWorkerReadyzCheckers
    Adapter-->>Boot: []DependencyChecker

    Note over Boot,SelfProbe: Self-probe — flag começa false
    Boot->>State: SetSelfProbe(false)
    Boot->>SelfProbe: RunSelfProbe(ctx, checkers, logger)
    par Probe paralelo
        SelfProbe->>SelfProbe: goroutine por checker (com PerDepTimeout)
    end
    alt todos up/skipped/n/a
        SelfProbe->>State: SetSelfProbe(true)
        SelfProbe-->>Boot: nil
    else algum down/degraded
        SelfProbe->>State: mantém SelfProbe(false)
        SelfProbe-->>Boot: error (advisory)
        Note over Boot: log error — pod fica vivo,<br/>kubelet decide via /health
    end

    Boot->>Server: NewServer(...) ou NewHealthServer(...)
    Server-->>Boot: pronto

    Note over Boot,Main: Devolve Service para o main rodar
    Boot-->>Main: *Service
    Main->>Server: Run() (sob commons.Launcher)
    Server->>K8s: aceitando GET /readyz e /health

    K8s->>Server: GET /readyz (depois de algum tempo)
    Server-->>K8s: 200 ou 503
```

**Pontos-chave:**

- O TLS gate está **antes** das `init*`. Se Mongo plaintext em SaaS mode, conexões nunca abrem.
- Self-probe **não** crasha em falha. Pod fica vivo para `/health` servir 503.
- Worker e manager seguem o mesmo padrão — só mudam os `init*` chamados.

---

## Diagrama 2 — `GET /readyz` no caminho feliz

A request normal: pod está saudável, todos os deps respondem rápido. Foco no fan-out paralelo.

```mermaid
sequenceDiagram
    autonumber
    participant K8s as K8s readinessProbe
    participant Routes as Routes / Fiber
    participant Handler as readyz.Handler
    participant State as State
    participant Mongo as MongoChecker
    participant Redis as RedisChecker
    participant TM as TMChecker
    participant Metrics as Metrics
    participant DepMongo as MongoDB
    participant DepRedis as Redis
    participant DepTM as TenantManager

    K8s->>Routes: GET /readyz
    Routes->>Handler: Fiber()(ctx)
    Handler->>State: IsDraining()
    State-->>Handler: false

    Note over Handler: ctx com per-dep deadline,<br/>fan-out goroutines

    par Probes em paralelo
        Handler->>Mongo: Check(ctx)
        Mongo->>DepMongo: Ping(ctx)
        DepMongo-->>Mongo: ok (50ms)
        Mongo-->>Handler: {status:up, latency:50}
    and
        Handler->>Redis: Check(ctx)
        Redis->>DepRedis: PING
        DepRedis-->>Redis: PONG (5ms)
        Redis-->>Handler: {status:up, latency:5}
    and
        Handler->>TM: Check(ctx)
        TM->>DepTM: GetActiveTenantsByService
        DepTM-->>TM: []tenants (80ms)
        TM-->>Handler: {status:up, latency:80}
    end

    Note over Handler,Metrics: Para cada probe — emite métricas
    Handler->>Metrics: emitCheckDuration(dep, status, elapsed)
    Handler->>Metrics: emitCheckStatus(dep, status)

    Handler->>Handler: aggregateStatus(checks)
    Note over Handler: todos up → "healthy"

    Handler-->>Routes: ReadyzResponse{status:healthy, ...}
    Routes-->>K8s: HTTP 200<br/>{"status":"healthy", "checks":{...}}
```

**Pontos-chave:**

- Probes rodam **em paralelo**. Tempo total ≈ tempo do checker mais lento, não a soma.
- `aggregateStatus` é a regra única: qualquer `down`/`degraded` derruba para `unhealthy` (HTTP 503). Status `up`, `skipped` e `n/a` contam como saudáveis.
- Métricas são emitidas **incondicionalmente** por probe — fundamental para `rate()` queries no Prometheus funcionarem.

---

## Diagrama 3 — `GET /readyz` durante drain

O pod já recebeu SIGTERM. O handler short-circuita: nem chega a chamar os checkers.

```mermaid
sequenceDiagram
    autonumber
    participant K8s as K8s readinessProbe
    participant Routes as Routes / Fiber
    participant Handler as readyz.Handler
    participant State as State
    participant Metrics as Metrics

    Note over State: SetDraining(true) já foi<br/>chamado pela goroutine de SIGTERM

    K8s->>Routes: GET /readyz
    Routes->>Handler: Fiber()(ctx)
    Handler->>State: IsDraining()
    State-->>Handler: true

    Note over Handler: Curto-circuito — não roda probes

    Handler->>Metrics: emitCheckDuration("draining", down, 0)
    Handler->>Metrics: emitCheckStatus("draining", down)

    Handler-->>Routes: ReadyzResponse{<br/>status:unhealthy,<br/>checks:{draining:{status:down, reason:"graceful drain"}}<br/>}
    Routes-->>K8s: HTTP 503

    Note over K8s: kube-proxy remove o pod<br/>do Service em ~2s
```

**Pontos-chave:**

- `IsDraining()` é leitura de `atomic.Bool`. Custo desprezível, seguro para hot path.
- Métricas continuam sendo emitidas com dep `"draining"` — dashboards conseguem rastrear rolling deploys.
- Resposta tem shape idêntico ao modo normal, só com um único check sintético `"draining"`.

---

## Diagrama 4 — `GET /readyz/tenant/:id`

O endpoint per-tenant. Tem duas fases: validação da tenant + probes per-tenant.

```mermaid
sequenceDiagram
    autonumber
    participant Caller as Operador / dashboard
    participant Routes as Routes / Fiber
    participant TenantH as TenantFiberHandler<br/>(tenant_handler.go)
    participant State as State
    participant TMClient as TMClient
    participant Probers as TenantProbers<br/>(tenant_probers.go)
    participant Mongo as TenantMongoChecker
    participant Rabbit as TenantRabbitMQChecker
    participant Metrics as Metrics
    participant DepMongo as Mongo da tenant
    participant DepRabbit as vhost AMQP da tenant

    Caller->>Routes: GET /readyz/tenant/abc-123
    Routes->>TenantH: Fiber()(ctx, id="abc-123")

    TenantH->>State: IsDraining()
    State-->>TenantH: false

    Note over TenantH,TMClient: Fase 1 — validar tenant existe
    TenantH->>TMClient: GetActiveTenantsByService<br/>(ctx 1s timeout)
    alt circuit breaker open
        TMClient-->>TenantH: ErrCircuitBreakerOpen
        TenantH-->>Routes: 503 {"error":"tenant manager circuit breaker open"}
    else lista de tenants
        TMClient-->>TenantH: []tenants
        alt id não está na lista
            TenantH-->>Routes: 404 {"error":"tenant not found"}
        end
    end

    Note over TenantH,Probers: Fase 2 — probes per-tenant
    TenantH->>TenantH: ctx = ContextWithTenantID(ctx, id)

    par Probers em paralelo (PerDepTimeout)
        TenantH->>Mongo: CheckForTenant(ctx, id)
        Mongo->>Probers: GetDatabaseForTenant(ctx, id)
        Probers->>DepMongo: Ping(readpref.Primary)
        DepMongo-->>Mongo: ok
        Mongo-->>TenantH: {status:up, latency:42}
    and
        TenantH->>Rabbit: CheckForTenant(ctx, id)
        Rabbit->>DepRabbit: GetChannel(ctx, id)
        DepRabbit-->>Rabbit: *amqp.Channel
        Rabbit->>Rabbit: ch.Close()
        Rabbit-->>TenantH: {status:up, latency:60}
    end

    TenantH->>Metrics: emitCheckDuration / emitCheckStatus<br/>(sem label de tenant_id — cardinalidade)

    TenantH->>TenantH: aggregateStatus(checks)
    TenantH-->>Routes: ReadyzResponse{<br/>status:healthy,<br/>tenant_id:"abc-123",<br/>checks:{mongodb,rabbitmq}<br/>}
    Routes-->>Caller: HTTP 200
```

**Pontos-chave:**

- **Fase 1 (validação)** tem timeout próprio de 1s, igual ao `PerDepTimeout("tenant_manager")`.
- **Fase 2 (probes)** usa o mesmo padrão paralelo do `/readyz` global.
- Resposta carrega `tenant_id` extra. Métricas **não** ganham label de tenant_id — protege cardinalidade do Prometheus.
- Probe de Rabbit abre channel **e fecha imediatamente** — só prova conectividade, não segura recurso.

---

## Diagrama 5 — Drain completo (SIGTERM)

A sequência mais delicada: como o pod sai limpo do tráfego sem dropar request. Mostro o caminho do **manager**; worker é análogo (substitua `Server.drainLoop` por `MultiQueueConsumer.Run`'s SIGTERM goroutine, e `ServerManager` por `commons.Launcher`).

```mermaid
sequenceDiagram
    autonumber
    participant Kubelet as kubelet
    participant Pod as Process (Go)
    participant DrainLoop as drainLoop<br/>(server.go)
    participant State as State
    participant SrvMgr as ServerManager<br/>(lib-commons)
    participant Hooks as Shutdown hooks
    participant Handler as readyz.Handler
    participant K8sProxy as kube-proxy
    participant K8sProbe as readinessProbe (k8s)
    participant InFlight as Request em curso

    Note over Kubelet,Pod: Pod marcado para terminar
    Kubelet->>Pod: SIGTERM

    Pod->>DrainLoop: <-sigs (goroutine bloqueada acorda)
    DrainLoop->>State: SetDraining(true)
    Note over State: drainingState atomic = true

    par Próximas requests
        K8sProbe->>Handler: GET /readyz
        Handler->>State: IsDraining()
        State-->>Handler: true
        Handler-->>K8sProbe: HTTP 503 (draining)
        K8sProbe->>K8sProxy: pod não-ready
        K8sProxy->>K8sProxy: remove pod do Service<br/>(em ~1-2s)
    and Drain delay
        DrainLoop->>DrainLoop: time.After(drainDelay = 12s)
    and Requests em curso
        InFlight->>Handler: continua sendo processada
        Note over InFlight: request termina normalmente
    end

    Note over DrainLoop: 12s passaram

    DrainLoop->>SrvMgr: close(shutdownCh)

    Note over SrvMgr: Agora SrvMgr começa shutdown real
    SrvMgr->>SrvMgr: fecha listener HTTP
    Note over SrvMgr: novas conexões rejeitadas;<br/>conexões abertas finalizam graceful

    SrvMgr->>Hooks: invoca shutdown hooks
    Hooks->>Hooks: licenseShutdown.Terminate()
    Hooks->>Hooks: rdb.Close() (readyzRedisClient)
    Hooks->>Hooks: tmMongoManager.Close()

    SrvMgr-->>Pod: shutdown complete
    Pod-->>Kubelet: exit code 0
```

**Pontos-chave (esta é a sequência mais importante para entender):**

- **Por que `SetDraining(true)` antes de fechar o listener?** Porque entre o flag virar e o k8s remover do Service passam ~1-2s. Se o listener já estivesse fechado, novas conexões nesse intervalo dariam connection refused.
- **Por que dormir 12s?** É o sync window padrão do kube-proxy. Damos tempo para a remoção do Service propagar a todos os nós antes de fechar listeners.
- **Quem dispara o `close(shutdownCh)`?** O `drainLoop` controla, **não** o `ServerManager`. O ServerManager bloqueia esperando esse channel via `WithShutdownChannel`. Esse é o truque que dá ao manager controle sobre a ordem.
- **Hooks rodam por último.** Liberar Redis dedicado, license, etc. — só depois do listener fechado para garantir que nada ainda dependia desses recursos.

### Versão worker do drain

A diferença do worker é que **o consumer também participa**:

```mermaid
sequenceDiagram
    autonumber
    participant Kubelet as kubelet
    participant Consumer as MultiQueueConsumer<br/>(consumer.go)
    participant State as State
    participant Health as HealthServer
    participant Launcher as commons.Launcher
    participant K8sProbe as readinessProbe
    participant Rabbit as RabbitMQ

    Kubelet->>Consumer: SIGTERM (interceptado pelo signal.Notify)

    Consumer->>State: SetDraining(true)

    par
        K8sProbe->>Health: GET /readyz
        Health->>State: IsDraining()
        State-->>Health: true
        Health-->>K8sProbe: 503 (draining)
        Note over K8sProbe: pod sai do Service
    and
        Consumer->>Consumer: time.After(drainDelay)
    and
        Consumer->>Rabbit: continua processando<br/>mensagens em curso
        Rabbit-->>Consumer: ack/nack normal
    end

    Note over Consumer: Drain delay completo
    Consumer->>Consumer: cancel() do ctx do consumer

    Consumer->>Rabbit: ConsumerLoop sai (ctx.Done)
    Note over Consumer,Rabbit: para de puxar novas mensagens

    Consumer->>Launcher: graceful shutdown completo
    Launcher->>Health: também encerra
    Health-->>Launcher: ok
    Launcher-->>Kubelet: exit
```

A diferença chave: **o cancel do ctx só vem depois do drain delay**. Se viesse antes, mensagens em curso seriam abortadas.

---

## Apêndice — Fluxos resumidos lado a lado

| Cenário | Quem flipa estado | Quem responde HTTP | Quem fecha recursos |
|---|---|---|---|
| Boot OK | `RunSelfProbe` flipa `selfProbeOK=true` | Server aceita requests | — |
| Boot com dep down | `selfProbeOK` fica `false` | `/health` 503 indefinido | — |
| `/readyz` saudável | nenhum | `Handler` agrega → 200 | — |
| `/readyz` com dep down | nenhum | `Handler` agrega → 503 | — |
| Drain (manager) | `drainLoop` flipa `drainingState=true` | `Handler` curto-circuita → 503 | shutdown hooks após 12s |
| Drain (worker) | goroutine SIGTERM em `consumer.go` flipa | `Handler` no `HealthServer` curto-circuita | consumer cancela após 12s |
| `/readyz/tenant/:id` ok | nenhum | `TenantFiberHandler` (2 fases) → 200 | — |
| `/readyz/tenant/:id` tenant inexistente | nenhum | retorno antes da fase 2 → 404 | — |

---

## Para fixar

Tente desenhar mentalmente o que acontece nestes cenários (ou rode o código com prints adicionados):

1. SIGTERM chega no manager **enquanto** uma request `/readyz` já está executando. O que acontece com essa request? (resposta: continua até o fim — só requests que **chegarem depois** veem `IsDraining()=true`.)

2. MongoDB cai durante 5 segundos. O `/readyz` reporta `down` em quanto tempo? (resposta: até 2s — o `PerDepTimeout("mongodb")`.)

3. Você reinicia o Tenant Manager. O circuit breaker do `tmclient` abre. O que o `/readyz` da fetcher mostra para o checker `tenant_manager`? (resposta: `{status:down, breaker_state:"open", error:"circuit breaker open"}`.)

4. O `selfProbeOK` está `false` mas todos os deps estão saudáveis no momento. Qual o status de `/health` e `/readyz`? (resposta: `/health` = 503 — não saiu do estado inicial. `/readyz` = 200 — não depende do self-probe.)

Esses cenários ajudam a internalizar a separação entre liveness (`/health`), readiness (`/readyz`) e drain — três conceitos que parecem o mesmo mas servem propósitos distintos.
