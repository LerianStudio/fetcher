# Detalhamento por arquivo — `components/manager/internal/` e `components/worker/`

Documento didático cobrindo, **arquivo a arquivo**, todas as alterações e novos arquivos dentro dessas duas pastas nos commits `dcd5fc2`, `2d38baf`, `f7d1754`.

> Pré-requisito: leia primeiro `docs/readyz-walkthrough.md` para entender a arquitetura geral. Este documento assume que você já sabe **o que** o `/readyz` faz; aqui detalhamos **como** cada arquivo dos componentes implementa isso.

---

## Como ler este documento

Cada seção segue o mesmo padrão:

- **Status** — `NEW` ou `MODIFIED`.
- **Responsabilidade** — em uma frase.
- **Estruturas e funções-chave** — o que existe no arquivo.
- **Walkthrough** — explicação por bloco, focando no que não é óbvio.
- **Conexões** — quais outros arquivos dependem deste / dos quais este depende.

Arquivos de teste estão fora do escopo deste documento.

---

# Parte I — Manager (`components/manager/internal/`)

O manager é um servidor HTTP Fiber que roda na porta 4006 (`SERVER_PORT`). A integração com `/readyz` foi feita reutilizando o servidor primário — não há micro-servidor separado.

## 1. `adapters/http/in/routes.go` — MODIFIED

**Responsabilidade:** registrar todas as rotas HTTP do manager num `*fiber.App`, na ordem certa.

### O que mudou

A função `NewRoutes` ganhou três parâmetros:

```go
readyzHandler       fiber.Handler
readyzTenantHandler fiber.Handler
metricsHandler      fiber.Handler
```

E três rotas foram montadas **antes** de qualquer middleware de auth:

```go
f.Get("/health", readyz.HealthHandler())
if readyzHandler != nil {
    f.Get("/readyz", readyzHandler)
}
if readyzTenantHandler != nil {
    f.Get("/readyz/tenant/:id", readyzTenantHandler)
}
if metricsHandler != nil {
    f.Get("/metrics", metricsHandler)
}
```

### Walkthrough do que mudou

**Por que os handlers vêm como parâmetros, e não construídos dentro de `NewRoutes`?**

Inversão de dependência. O `routes.go` está na camada de adapters HTTP — ele sabe sobre Fiber, sobre middlewares, sobre paths. Ele **não** deve saber como construir um `*readyz.Handler` (que precisa de mongoClient, redisClient, rabbit adapter, etc.). Quem sabe construir essas coisas é o `bootstrap/`. Então: `bootstrap/` injeta os handlers prontos.

**Por que os parâmetros são `fiber.Handler` em vez de `*readyz.Handler`?**

Para que o adapter HTTP **não importe** o pacote `readyz` desnecessariamente. Recebendo `fiber.Handler` (interface da própria Fiber), o `routes.go` fica desacoplado da implementação do readyz — amanhã alguém pode testar com um stub de `fiber.Handler` sem mockar tudo do readyz.

**Por que o teste `if readyzHandler != nil`?**

Cinto e suspensório — é defesa contra um bootstrap que esquece de injetar. Se for nil, a rota simplesmente não é montada. Em produção isso seria um bug; em testes parciais (que não montam todas as deps) evita NPE.

**`/health` é montado sem o `if`. Por quê?**

`readyz.HealthHandler()` é uma função pura que apenas lê `selfProbeOK` (atomic bool). Não depende de nada injetável, então pode ser construída ali mesmo. É o caso "construir é trivial, não vale ter parâmetro".

**Posição relativa ao auth middleware:**

```
f.Use(http.WithRecover(...))
f.Use(tlMid.WithTelemetry(tl))
f.Use(cors.New())
f.Use(commonsHttp.WithHTTPLogging(...))

f.Get("/swagger/*", ...)
f.Get("/health", ...)         ← AQUI, sem auth
f.Get("/readyz", ...)
...

// Mais abaixo:
f.Post("/v1/management/connections", auth.Authorize(...), ...)
```

A ordem importa. As rotas de probe são GETs simples mountados antes das rotas autenticadas. Como Fiber faz match na ordem de registro, qualquer GET em `/readyz` cai na linha do probe — sem nunca passar por `auth.Authorize`.

### Conexões

- **Recebe handlers de:** `bootstrap/config.go` (em `assembleService`).
- **Usa:** `pkg/bootstrap/readyz/registration.go` (`HealthHandler`).

---

## 2. `bootstrap/config.go` — MODIFIED

**Responsabilidade:** orquestra o bootstrap completo do manager — carrega config, inicializa Mongo, RabbitMQ, Redis, crypto, multi-tenant, e por fim entrega um `*Service` pronto para rodar.

### O que foi adicionado

#### 2.1. Novos campos no struct `Config`

```go
DeploymentMode      string `env:"DEPLOYMENT_MODE" default:"local"`
ReadyzDrainDelaySec int    `env:"READYZ_DRAIN_DELAY_SEC" default:"12"`
```

`DeploymentMode` controla o portão de TLS SaaS. `ReadyzDrainDelaySec` é o tempo de drain — quanto o servidor espera depois de SIGTERM antes de fechar listeners.

#### 2.2. Novos campos no `managerRepositories`

```go
type managerRepositories struct {
    connection  *connection.ConnectionMongoDBRepository
    job         *job.JobMongoDBRepository
    mongoClient *libMongo.Client   // ← novo
}
```

O `mongoClient` é o cliente bruto do lib-commons. Ele já era criado em `initMongoRepositories`, só não era retornado. Agora é exposto para que o `/readyz` possa rodar `Ping()` nele sem abrir uma segunda conexão.

#### 2.3. Novos campos no `managerPlatformDependencies`

Adicionados:
```go
rabbitMQAdapter   rabbitmq.Adapter      // ← circuit breaker visível ao readyz
tmClient          *tmclient.Client      // ← cliente TM compartilhado
tmMongoManager    *tmmongo.Manager      // ← per-tenant Mongo manager
tmRabbitMQManager *tmrabbitmq.Manager   // ← per-tenant Rabbit manager
readyzRedisClient *redis.Client         // ← Redis dedicado ao probe
```

Cada um é uma "amarra" passada do bootstrap para o readyz wiring. **Por que `readyzRedisClient` é separado?** Porque o cliente Redis usado pelo schema cache vem encapsulado num wrapper de fallback (`redis.NewCacheWithFallback`) que sempre devolve "ok" mesmo com Redis caído (cai no in-memory). Probar pelo wrapper esconderia falha real.

#### 2.4. Variável `validateSaaSTLSFn`

```go
var validateSaaSTLSFn = readyz.ValidateSaaSTLS
```

Uma variável de pacote que aponta para a função real. Por que não chamar `readyz.ValidateSaaSTLS` direto? Para permitir override em testes:

```go
// no _test.go:
validateSaaSTLSFn = func(cfg readyz.SaaSTLSConfig) error { return nil }
```

Esse padrão "var de pacote como seam de teste" é canônico em Go quando você precisa stub uma função sem usar interface. Repare que outras funções já usavam esse padrão (`loadConfigFn`, `initMongoRepositoriesFn`, etc.) — `validateSaaSTLSFn` se encaixa.

#### 2.5. Chamada de `validateSaaSTLSFn` em `InitServers`

```go
if err := validateSaaSTLSFn(buildSaaSTLSConfig(cfg, false)); err != nil {
    return nil, fmt.Errorf("tls enforcement failed: %w", err)
}

repositories, err := initMongoRepositoriesFn(ctx, cfg, logger)
```

A ordem é crítica. O `validateSaaSTLSFn` corre **antes** de `initMongoRepositoriesFn` — e antes de qualquer outra `init*`. Se alguém configurar `DEPLOYMENT_MODE=saas` com Mongo plaintext, o boot falha sem nem tentar conectar.

`hasS3=false` porque o manager não usa S3 (é dependência exclusiva do worker).

#### 2.6. Função `assembleService` ganhou wiring de readyz

Trecho relevante:

```go
readyzCfg := newReadyzConfig(cfg)
globalCheckers := buildManagerReadyzCheckers(cfg, repositories, platformDependencies)

runManagerSelfProbe(context.Background(), logger, globalCheckers)

readyzHandler := readyz.NewHandler(readyzCfg, globalCheckers...).Fiber()
tenantFiberHandler := buildManagerTenantHandler(readyzCfg, cfg, platformDependencies)

httpApp := in2.NewRoutes(
    ...
    readyzHandler,
    tenantFiberHandler,
    readyz.NewMetricsHandler(),
)
```

Sequência de leitura:

1. `newReadyzConfig(cfg)` — converte o `*Config` do manager para `*readyz.Config` (definido depois).
2. `buildManagerReadyzCheckers(...)` — fábrica que monta a lista de probers (definida em `readyz_adapters.go`).
3. `runManagerSelfProbe(...)` — roda os probers uma vez no boot, ajusta o flag `selfProbeOK`. Mesmo se falhar, **não** crasha o pod.
4. `readyz.NewHandler(...)` — constrói o handler com config + checkers.
5. `buildManagerTenantHandler(...)` — constrói o handler do `/readyz/tenant/:id` (real ou disabled).
6. `in2.NewRoutes(...)` — injeta os três handlers na fábrica de rotas.

#### 2.7. Shutdown hooks de readyz

```go
if platformDependencies.readyzRedisClient != nil {
    rdb := platformDependencies.readyzRedisClient
    shutdownHooks = append(shutdownHooks, func(context.Context) error {
        _ = rdb.Close()
        return nil
    })
}

if platformDependencies.tmMongoManager != nil {
    mgr := platformDependencies.tmMongoManager
    shutdownHooks = append(shutdownHooks, func(ctx context.Context) error {
        _ = mgr.Close(ctx)
        return nil
    })
}
```

Recursos criados especificamente para o readyz precisam ser liberados no shutdown. O closure captura por valor (`rdb := ...`) para isolar do loop e evitar pegar a última referência.

#### 2.8. `buildSaaSTLSConfig`

```go
func buildSaaSTLSConfig(cfg *Config, hasS3 bool) readyz.SaaSTLSConfig {
    return readyz.SaaSTLSConfig{
        DeploymentMode:      cfg.DeploymentMode,
        MongoURI:            buildMongoSource(cfg),
        RedisURL:            readyz.ComposeRedisURL(cfg.RedisHost, cfg.RedisPort, cfg.RedisTLS),
        MultiTenantRedisURL: readyz.ComposeRedisURL(cfg.MultiTenantRedisHost, cfg.MultiTenantRedisPort, cfg.MultiTenantRedisTLS),
        RabbitMQURL:         buildRabbitMQSource(cfg),
        S3Endpoint:          "",
        TenantManagerURL:    cfg.MultiTenantURL,
        MultiTenantEnabled:  cfg.MultiTenantEnabled,
        HasS3:               hasS3,
        AllowInsecureHTTPTM: cfg.MultiTenantAllowInsecureHTTP,
    }
}
```

Tradutor `Config → SaaSTLSConfig`. Importante: usa as **mesmas funções** (`buildMongoSource`, `buildRabbitMQSource`) que serão usadas para abrir conexões reais. Garante que o validador inspeciona exatamente o que será dialed depois.

#### 2.9. `newReadyzConfig`

```go
func newReadyzConfig(cfg *Config) *readyz.Config {
    if cfg == nil {
        return readyz.LoadConfig()
    }

    drain := time.Duration(cfg.ReadyzDrainDelaySec) * time.Second
    if cfg.ReadyzDrainDelaySec <= 0 {
        drain = 12 * time.Second
    }

    mode := cfg.DeploymentMode
    if mode == "" {
        mode = readyz.DeploymentModeLocal
    }

    version := cfg.OtelServiceVersion
    if version == "" {
        version = "unknown"
    }

    return &readyz.Config{
        DeploymentMode: mode,
        DrainDelay:     drain,
        Version:        version,
    }
}
```

Adapter de tipos. Aplica os mesmos defaults que `readyz.LoadConfig` aplicaria a partir do env. **Por que não chamar `readyz.LoadConfig()` direto?** Porque o manager já leu o env e tem os valores em `*Config`. Re-ler do env seria redundante e poderia ter resultado divergente se o env mudou entre uma leitura e outra.

#### 2.10. Self-probe wrapper

```go
const selfProbeTimeout = 15 * time.Second

var runManagerSelfProbe = func(ctx context.Context, logger libLog.Logger, checkers []readyz.DependencyChecker) {
    probeCtx, cancel := context.WithTimeout(ctx, selfProbeTimeout)
    defer cancel()

    if err := readyz.RunSelfProbe(probeCtx, checkers, logger); err != nil {
        if logger != nil {
            logger.Log(ctx, libLog.LevelError,
                "startup self-probe reported unhealthy deps; ...",
                libLog.Err(err),
            )
        }
    }
}
```

Wrapper simples, mas com duas decisões importantes:

1. **15s de timeout total** — cobre o pior caso (todos os deps no PerDepTimeout=2s, mais alguma folga). Se um dep ficar pendurado, o self-probe não trava o boot indefinidamente.
2. **Erro virou log, não retorno** — o boot continua. Quem leu o `pkg/bootstrap/readyz/selfprobe.go` já viu por quê: zero-panic policy, deixar o pod ficar 503 e o kubelet decidir.

`var = func(...)` em vez de `func` — outro seam de teste. Permite override em `_test.go`.

### Conexões

- **Lê de:** envs do manager, `pkg/bootstrap/readyz/`.
- **Entrega para:** `routes.go` (handlers prontos).
- **Auxilia:** `readyz_adapters.go` (consome `repositories` e `platformDependencies`).

---

## 3. `bootstrap/readyz_adapters.go` — NEW

**Responsabilidade:** ponte entre as deps concretas do manager e o pacote canônico `readyz`.

### Estruturas e funções

```go
type rabbitMQAdapterProbe struct {
    adapter pkgRabbitmq.Adapter
}

func newRabbitMQAdapterProbe(adapter pkgRabbitmq.Adapter) *rabbitMQAdapterProbe
func (r *rabbitMQAdapterProbe) State() readyz.BreakerState
func (r *rabbitMQAdapterProbe) Ping(ctx context.Context) error

func newReadyzRedisClient(cfg *Config) *redis.Client

func redisURLFromCfg(host, port string, useTLS bool) string
func multiTenantRedisURLFromCfg(cfg *Config) string

func applicationServiceName() string

func buildManagerReadyzCheckers(cfg *Config, repos *managerRepositories,
                                 plat *managerPlatformDependencies) []readyz.DependencyChecker

func buildManagerTenantHandler(readyzCfg *readyz.Config, cfg *Config,
                                plat *managerPlatformDependencies) fiber.Handler
```

### Walkthrough

#### 3.1. `rabbitMQAdapterProbe`

```go
func (r *rabbitMQAdapterProbe) State() readyz.BreakerState {
    if r == nil || r.adapter == nil {
        return readyz.BreakerClosed
    }

    switch r.adapter.CircuitBreakerState() {
    case pkgRabbitmq.CircuitClosed:
        return readyz.BreakerClosed
    case pkgRabbitmq.CircuitOpen:
        return readyz.BreakerOpen
    case pkgRabbitmq.CircuitHalfOpen:
        return readyz.BreakerHalfOpen
    default:
        return readyz.BreakerClosed
    }
}
```

Tradutor de enum. **Por que existe?** Porque `pkgRabbitmq.CircuitState` (do projeto fetcher) é um tipo, e `readyz.BreakerState` é outro. Eles **não** podem ser convertidos com cast (`readyz.BreakerState(r.adapter.CircuitBreakerState())`) porque o pacote `readyz` deliberadamente não importa `pkg/rabbitmq` — manter o `readyz` no fundo do grafo de dependências.

Note o `default: BreakerClosed`. Se o pkg/rabbitmq adicionar um `CircuitUnknown` no futuro, o readyz reporta como `closed` em vez de surtar — fail-safe.

```go
func (r *rabbitMQAdapterProbe) Ping(ctx context.Context) error {
    if r == nil || r.adapter == nil {
        return errAdapterNil
    }
    if err := ctx.Err(); err != nil {
        return err
    }
    if !r.adapter.IsHealthy() {
        return errNotHealthy
    }
    return nil
}
```

`Ping` é **passivo**. Lê `IsHealthy()` (uma flag mantida pelo channel watcher do adapter), não abre channel novo. Esse é um detalhe que o comentário do código preserva: abrir channel a cada `/readyz` esgotaria limite de channels do servidor RabbitMQ em pouco tempo.

#### 3.2. Erros sentinela

```go
var (
    errAdapterNil = &adapterError{msg: "rabbitmq adapter not initialized"}
    errNotHealthy = &adapterError{msg: "rabbitmq connection not healthy"}
)

type adapterError struct{ msg string }
func (e *adapterError) Error() string { return e.msg }
```

Tipos de erro próprios em vez de `errors.New`. Por quê? Para que callers possam fazer `errors.Is(err, errNotHealthy)` se precisarem. Em prática, o readyz só lê `err.Error()` e sanitiza. Mas o tipo dedicado dá flexibilidade futura sem custo.

#### 3.3. `newReadyzRedisClient`

```go
func newReadyzRedisClient(cfg *Config) *redis.Client {
    if cfg == nil || cfg.RedisHost == "" {
        return nil
    }
    port := cfg.RedisPort
    if port == "" {
        port = "6379"
    }
    opts := &redis.Options{
        Addr:     net.JoinHostPort(cfg.RedisHost, port),
        Password: cfg.RedisPassword,
        DB:       getRedisDB(cfg.RedisDB),
    }
    return redis.NewClient(opts)
}
```

Cliente Redis **dedicado** ao readyz. Já tratei o "porquê" em 4.3. Note `net.JoinHostPort` em vez de `host + ":" + port` — bracket de IPv6 correto.

#### 3.4. `buildManagerReadyzCheckers` — a fábrica

```go
func buildManagerReadyzCheckers(
    cfg *Config,
    repos *managerRepositories,
    plat *managerPlatformDependencies,
) []readyz.DependencyChecker {
    if cfg == nil {
        return nil
    }

    checkers := make([]readyz.DependencyChecker, 0, 6)

    // MongoDB
    if cfg.MultiTenantEnabled {
        mongoTLS, _ := readyz.DetectMongoTLS(buildMongoSource(cfg))
        checkers = append(checkers, readyz.NewNAChecker(
            "mongodb",
            "multi-tenant: see /readyz/tenant/:id",
            readyz.TLSPtr(mongoTLS || cfg.MongoTLSCACert != ""),
        ))
    } else if repos != nil && repos.mongoClient != nil {
        checkers = append(checkers, readyz.NewMongoClientChecker(
            repos.mongoClient, buildMongoSource(cfg),
        ))
    }

    // RabbitMQ
    if cfg.MultiTenantEnabled {
        checkers = append(checkers, readyz.NewNAChecker("rabbitmq", "...", ...))
    } else if plat != nil && plat.rabbitMQAdapter != nil {
        checkers = append(checkers, readyz.NewRabbitMQAdapterChecker(...))
    }

    // Redis (schema cache)
    if plat != nil && plat.readyzRedisClient != nil {
        client := plat.readyzRedisClient
        checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
            "redis",
            func(ctx context.Context) error { return client.Ping(ctx).Err() },
            redisURLFromCfg(cfg.RedisHost, cfg.RedisPort, cfg.RedisTLS),
        ))
    }

    // Tenant Manager (apenas em MT)
    if cfg.MultiTenantEnabled && plat != nil {
        checkers = append(checkers, readyz.NewTenantManagerClientChecker(
            plat.tmClient,
            applicationServiceName(),
            cfg.MultiTenantURL,
            true,
        ))
    }

    return checkers
}
```

Função pura — recebe config e deps, devolve a lista. Sem efeito colateral, fácil de testar.

**Padrão importante:** cada bloco trata um dep e usa `if/else if` para escolher entre real e NAChecker. Adicionar um novo dep é só mais um bloco. Não há roteamento dinâmico, `switch`, ou registry — é explícito por design.

**Por que `readyz.TLSPtr(mongoTLS || cfg.MongoTLSCACert != "")`?**

Em multi-tenant, o readyz global devolve NAChecker para Mongo, mas precisa surfaçar a postura TLS. Duas fontes possíveis:

- Detecção pela URI: `mongodb+srv` ou `?tls=true` → TLS verdadeiro.
- Presença de CA cert: legado de operadores que configuravam CA explícito.

Faz `OR` lógico. Se algum dos dois for verdadeiro, reporta TLS habilitado. Esse é o tipo de detalhe que vai certo nos primeiros 99% dos casos e quebra no 100º.

#### 3.5. `buildManagerTenantHandler`

```go
func buildManagerTenantHandler(
    readyzCfg *readyz.Config,
    cfg *Config,
    plat *managerPlatformDependencies,
) fiber.Handler {
    if !cfg.MultiTenantEnabled || plat == nil || plat.tmClient == nil ||
        plat.tmMongoManager == nil || plat.tmRabbitMQManager == nil {
        return readyz.NewDisabledTenantHandler()
    }

    th := readyz.NewTenantHandler(
        readyzCfg,
        plat.tmClient,
        applicationServiceName(),
        readyz.NewTenantMongoChecker(plat.tmMongoManager),
        readyz.NewTenantRabbitMQChecker(plat.tmRabbitMQManager),
    )

    return th.Fiber()
}
```

Defesa em profundidade: cinco condições precisam ser todas verdadeiras para o handler real ser construído. Qualquer uma falsa cai no DisabledHandler (que devolve 400). Isso é mais seguro que `if cfg.MultiTenantEnabled` solitário — protege contra bugs em outras partes do bootstrap que zeram um manager.

### Conexões

- **Importa de:** `pkg/bootstrap/readyz/`, `pkg/rabbitmq/`.
- **Consumido por:** `bootstrap/config.go`.

---

## 4. `bootstrap/server.go` — MODIFIED

**Responsabilidade:** wrap do servidor HTTP — orquestra startup, shutdown, drain.

### O que foi adicionado

#### 4.1. `serverNotifySignals`

```go
var serverNotifySignals = signal.Notify
```

Variável de pacote apontando para `signal.Notify`. Outro seam de teste — permite injetar canal sintético.

#### 4.2. Novo campo `drainDelay`

```go
type Server struct {
    app           *fiber.App
    serverAddress string
    license       licenseTerminator
    logger        libCommonsLog.Logger
    telemetry     libCommonsOtel.Telemetry
    shutdownHooks []func(context.Context) error
    drainDelay    time.Duration  // ← novo
}
```

Quanto tempo dormir após SIGTERM antes de fechar listeners.

#### 4.3. `resolveServerDrainDelay`

```go
func resolveServerDrainDelay(cfg *Config) time.Duration {
    switch {
    case cfg == nil || cfg.ReadyzDrainDelaySec == 0:
        return 12 * time.Second
    case cfg.ReadyzDrainDelaySec < 0:
        return time.Second
    default:
        return time.Duration(cfg.ReadyzDrainDelaySec) * time.Second
    }
}
```

Mesmas regras de clamp que `readyz.LoadConfig`. **Por que clampar negativos para 1s e não para 0?** Porque drain de 0s = não drenar = derrubar conexões abruptamente. Pelo menos 1s dá margem para o kube-proxy.

Defina como local em vez de importar do pacote `readyz` para não importar uma dep só por essa função pequena.

#### 4.4. `Run` reescrito — o coração do drain

```go
func (s *Server) Run(l *libCommons.Launcher) error {
    _ = l

    shutdownCh := make(chan struct{})

    sig := make(chan os.Signal, 1)
    serverNotifySignals(sig, os.Interrupt, syscall.SIGTERM)

    drainCtx, drainCancel := context.WithCancel(context.Background())
    defer drainCancel()

    go s.drainLoop(drainCtx, sig, shutdownCh)

    manager := libCommonsServer.NewServerManager(nil, &s.telemetry, s.logger).
        WithHTTPServer(s.app, s.serverAddress).
        WithShutdownChannel(shutdownCh)
    ...
    manager.StartWithGracefulShutdown()
    signal.Stop(sig)
    drainCancel()

    return nil
}
```

A novidade está no padrão: **interceptar SIGTERM antes do ServerManager**. O ServerManager do lib-commons, se deixado sozinho, instala seu próprio handler de SIGTERM. Mas a sequência de drain dele é ruim para nós: ele fecha o listener e **depois** roda os hooks. Tarde demais — o `/readyz` precisa flipar para 503 **antes** do listener fechar.

Truque: passa um `shutdownCh` próprio via `WithShutdownChannel`. O ServerManager bloqueia nesse canal em vez de instalar handler de signal. Aí nós controlamos a sequência:

1. Goroutine `drainLoop` espera SIGTERM.
2. Quando chega, flipa `readyz.SetDraining(true)`.
3. Dorme `drainDelay` (12s default).
4. Fecha `shutdownCh` — só agora o ServerManager começa a fechar listener.

#### 4.5. `drainLoop`

```go
func (s *Server) drainLoop(ctx context.Context, sig <-chan os.Signal, shutdownCh chan<- struct{}) {
    select {
    case <-sig:
    case <-ctx.Done():
        close(shutdownCh)
        return
    }

    readyz.SetDraining(true)

    if s.logger != nil {
        s.logger.Log(ctx, libCommonsLog.LevelInfo, ...)
    }

    if s.drainDelay > 0 {
        select {
        case <-time.After(s.drainDelay):
        case <-ctx.Done():
        }
    }

    close(shutdownCh)
}
```

Estado-máquina de 3 fases:

1. **Espera de sinal:** bloqueia até SIGTERM ou ctx cancel.
2. **Flag de drain:** flipa `readyz.SetDraining(true)`.
3. **Sleep cancelável:** dorme `drainDelay`, mas pode ser interrompido pelo ctx.

`ctx.Done()` no select garante que se alguém cancelar o context (tipo, em testes), a goroutine não fica pendurada.

`close(shutdownCh)` no final é o sinal para o ServerManager começar shutdown real.

### Conexões

- **Importa:** `pkg/bootstrap/readyz/`.
- **Construído por:** `bootstrap/config.go` (em `assembleService`).

---

# Parte II — Worker (`components/worker/`)

O worker tem uma diferença estrutural: **ele não tem servidor HTTP primário**. Ele consome de RabbitMQ. Para suportar `/readyz`, foi criado um **micro-servidor dedicado** numa porta separada (`HEALTH_PORT`, default 4007).

## 5. `internal/adapters/rabbitmq/consumer.rabbitmq.go` — MODIFIED

**Responsabilidade:** wrapper de `rabbitmq.Adapter` com suporte a múltiplas filas.

### O que foi adicionado

```go
// Adapter exposes the underlying adapter so /readyz can inspect the
// circuit-breaker state and liveness without reaching into unexported
// fields.
func (c *ConsumerRoutes) Adapter() rabbitmq.Adapter {
    if c == nil {
        return nil
    }
    return c.adapter
}
```

Um getter que expõe o adapter interno. **Por que era necessário?** O `readyz_adapters.go` precisa do adapter para construir o `RabbitMQAdapterChecker`. Sem o getter, teria que reflectir ou expor o campo. O getter é a opção limpa.

Note o `if c == nil { return nil }` — getter nil-safe. Permite chamadas em ponteiros nil sem panic, útil em paths de teste.

### Conexões

- **Consumido por:** `components/worker/internal/bootstrap/readyz_adapters.go` (em `newWorkerReadyzDepsST`).

---

## 6. `internal/bootstrap/config.go` — MODIFIED

**Responsabilidade:** bootstrap completo do worker.

### O que foi adicionado (alto nível)

Estrutura paralela ao manager, mas com diferenças:

- Worker tem **S3** como dep direta. Manager não.
- Worker tem **multi_tenant_redis** como única Redis. Manager tem schema cache + multi_tenant_redis.
- Worker tem **dois caminhos de bootstrap** explícitos: single-tenant e multi-tenant. Manager tem if/else inline.

#### 6.1. Novos campos no `Config`

```go
DeploymentMode      string `env:"DEPLOYMENT_MODE" default:"local"`
HealthPort          int    `env:"HEALTH_PORT" default:"4007"`
ReadyzDrainDelaySec int    `env:"READYZ_DRAIN_DELAY_SEC" default:"12"`
```

`HealthPort` é exclusivo do worker — manager reusa `SERVER_PORT`.

#### 6.2. `validateSaaSTLSFn`

```go
var validateSaaSTLSFn = readyz.ValidateSaaSTLS
```

Mesmo padrão do manager.

#### 6.3. Chamada em `InitWorker`

```go
if err := validateSaaSTLSFn(buildSaaSTLSConfig(cfg, true)); err != nil {
    return nil, fmt.Errorf("tls enforcement failed: %w", err)
}

storageRepository, err := initStorageRepository(ctx, cfg)
```

Note `hasS3=true`. Worker tem S3 como dep, então o portão precisa validar.

A ordem é: TLS gate **antes** de qualquer `init*` — incluindo `initStorageRepository`. Se S3 estiver plaintext (`http://...`) em SaaS mode, o boot para sem nem tentar criar o cliente AWS.

#### 6.4. Bifurcação multi-tenant

```go
if cfg.MultiTenantEnabled && cfg.MultiTenantURL != "" {
    mtConsumer, tmClient, mtCleanup, mtErr := initMultiTenantStack(...)
    ...
    readyzDeps := newWorkerReadyzDepsMT(cfg, mongoConnection, storageRepository,
                                         tmClient, mongoManager, rabbitMQManager)
    runWorkerSelfProbe(ctx, logger, buildWorkerReadyzCheckers(readyzDeps))

    return &Service{
        MultiQueueConsumer: multiQueueConsumer,
        Logger:             logger,
        licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
        mtCleanup:          mtCleanup,
        healthServer:       NewHealthServer(cfg, logger, telemetry, readyzDeps),
        readyzCloser:       readyzDeps.close,
    }, nil
}

multiQueueConsumer, consumerRoutes, err := initSingleTenantRabbitMQ(...)
...
readyzDeps := newWorkerReadyzDepsST(cfg, mongoConnection, storageRepository, consumerRoutes)
runWorkerSelfProbe(ctx, logger, buildWorkerReadyzCheckers(readyzDeps))

return &Service{
    MultiQueueConsumer: multiQueueConsumer,
    Logger:             logger,
    licenseShutdown:    licenseClient.GetLicenseManagerShutdown(),
    healthServer:       NewHealthServer(cfg, logger, telemetry, readyzDeps),
    readyzCloser:       readyzDeps.close,
}, nil
```

Dois retornos quase idênticos — diferenças mínimas:

- MT path tem `mtCleanup` (Redis pub/sub do event listener).
- ST path tem `consumerRoutes` (que contém o adapter que vai pro readyz).

Cada path constrói `readyzDeps` com sua própria fábrica (`newWorkerReadyzDepsMT` vs `newWorkerReadyzDepsST`) — a separação fica em `readyz_adapters.go`.

`runWorkerSelfProbe` corre **antes** do consumer começar a consumir mensagens. A ideia é: se algum dep crítico está fora, sinaliza 503 imediatamente em vez de processar mensagens em estado degradado.

#### 6.5. Helpers de URL

```go
func buildSaaSTLSConfig(cfg *Config, hasS3 bool) readyz.SaaSTLSConfig {
    return readyz.SaaSTLSConfig{
        DeploymentMode:      cfg.DeploymentMode,
        MongoURI:            buildWorkerMongoURI(cfg),
        RedisURL:            "",  // worker não tem Redis primário
        MultiTenantRedisURL: readyz.ComposeRedisURL(cfg.MultiTenantRedisHost, ...),
        RabbitMQURL:         buildWorkerRabbitMQURL(cfg),
        S3Endpoint:          cfg.ObjectStorageEndpoint,
        TenantManagerURL:    cfg.MultiTenantURL,
        MultiTenantEnabled:  cfg.MultiTenantEnabled,
        HasS3:               hasS3,
        AllowInsecureHTTPTM: cfg.MultiTenantAllowInsecureHTTP,
    }
}

func buildWorkerMongoURI(cfg *Config) string { ... }
func buildWorkerRabbitMQURL(cfg *Config) string { ... }
```

`buildWorkerMongoURI` e `buildWorkerRabbitMQURL` reproduzem **exatamente** o que `initMongoConnection` e `initSingleTenantRabbitMQ` fazem. Tem que ser idêntico — senão o validador inspeciona uma URL e o código abre outra.

`RedisURL: ""` é proposital: o worker não tem Redis primário (o schema cache vive no manager). O ValidateSaaSTLS pula `RedisURL` vazio com a regra "dep não configurado".

#### 6.6. Self-probe wrapper

```go
const workerSelfProbeTimeout = 15 * time.Second

var runWorkerSelfProbe = func(ctx context.Context, logger libLog.Logger, checkers []readyz.DependencyChecker) {
    probeCtx, cancel := context.WithTimeout(ctx, workerSelfProbeTimeout)
    defer cancel()

    if err := readyz.RunSelfProbe(probeCtx, checkers, logger); err != nil {
        if logger != nil {
            logger.Log(ctx, libLog.LevelError, ..., libLog.Err(err))
        }
    }
}
```

Espelho do manager. Mesmo padrão, mesmas decisões.

### Conexões

- **Importa de:** `pkg/bootstrap/readyz/`.
- **Constrói:** `Service`, `HealthServer`, deps via factories em `readyz_adapters.go`.

---

## 7. `internal/bootstrap/health_server.go` — NEW

**Responsabilidade:** micro-servidor HTTP do worker, serve `/health`, `/readyz`, `/readyz/tenant/:id`, `/metrics`.

### Estruturas e funções

```go
const defaultHealthPort = 4007
const defaultReadyzDrainDelay = 12 * time.Second

func defaultDrain(sec int) time.Duration

type HealthServer struct {
    app       *fiber.App
    addr      string
    logger    libLog.Logger
    telemetry *libOtel.Telemetry
}

func NewHealthServer(cfg *Config, logger libLog.Logger, telemetry *libOtel.Telemetry,
                     deps *workerReadyzDeps) *HealthServer
func (s *HealthServer) App() *fiber.App
func (s *HealthServer) Address() string
func (s *HealthServer) Run(_ *libCommons.Launcher) error

func newWorkerReadyzConfig(cfg *Config) *readyz.Config
```

### Walkthrough

#### 7.1. `defaultDrain`

```go
func defaultDrain(sec int) time.Duration {
    switch {
    case sec < 0:
        return time.Second
    case sec == 0:
        return defaultReadyzDrainDelay  // 12s
    default:
        return time.Duration(sec) * time.Second
    }
}
```

Mesma regra do manager. Espelho intencional para que worker e manager exibam o mesmo comportamento.

#### 7.2. `NewHealthServer` — construção do app Fiber

```go
func NewHealthServer(
    cfg *Config,
    logger libLog.Logger,
    telemetry *libOtel.Telemetry,
    deps *workerReadyzDeps,
) *HealthServer {
    readyzCfg := newWorkerReadyzConfig(cfg)
    checkers := buildWorkerReadyzCheckers(deps)
    handler := readyz.NewHandler(readyzCfg, checkers...)

    app := fiber.New(fiber.Config{
        DisableStartupMessage: true,
    })

    app.Get("/health", readyz.HealthHandler())
    app.Get("/readyz", handler.Fiber())
    app.Get("/readyz/tenant/:id", buildWorkerTenantHandler(readyzCfg, deps))
    app.Get("/metrics", readyz.NewMetricsHandler())

    return &HealthServer{
        app:       app,
        addr:      fmt.Sprintf(":%d", readyzCfg.HealthPort),
        logger:    logger,
        telemetry: telemetry,
    }
}
```

Sequência:

1. Constrói o config readyz.
2. Constrói os checkers via fábrica.
3. Constrói o handler com config + checkers.
4. Cria fiber.App vazio (sem CORS, sem auth, sem telemetry — é micro-servidor).
5. Registra 4 rotas.
6. Devolve struct com `addr` formatado de `readyzCfg.HealthPort`.

**Por que `addr := fmt.Sprintf(":%d", readyzCfg.HealthPort)`?** Porque `readyzCfg.HealthPort` já passou pelo clamp (0/negativo/>65535 → 4007 default). Usar o valor cru de `cfg.HealthPort` permitiria escutar em `:0` (porta aleatória) — péssimo para um probe.

`DisableStartupMessage: true` — Fiber por padrão imprime um banner enorme em stdout. Desnecessário para micro-servidor.

#### 7.3. `Run`

```go
func (s *HealthServer) Run(_ *libCommons.Launcher) error {
    if s.logger != nil {
        s.logger.Log(context.Background(), libLog.LevelInfo,
            fmt.Sprintf("worker health server listening on %s", s.addr))
    }

    manager := libCommonsServer.NewServerManager(nil, s.telemetry, s.logger).
        WithHTTPServer(s.app, s.addr)

    manager.StartWithGracefulShutdown()

    return nil
}
```

A assinatura aceita `*libCommons.Launcher` mas ignora — necessário para satisfazer a interface `commons.RunApp`. O `ServerManager` cuida de listen + graceful shutdown.

Note que **diferente do manager**, o worker não tem `drainLoop` próprio aqui — porque o drain é coordenado pelo `consumer.go` (vamos ver). O HealthServer só serve as rotas; quem vira o flag de drain é o consumer ao receber SIGTERM.

#### 7.4. `newWorkerReadyzConfig`

```go
func newWorkerReadyzConfig(cfg *Config) *readyz.Config {
    if cfg == nil {
        return readyz.LoadConfig()
    }

    mode := cfg.DeploymentMode
    if mode == "" {
        mode = readyz.DeploymentModeLocal
    }

    drain := defaultDrain(cfg.ReadyzDrainDelaySec)

    version := cfg.OtelServiceVersion
    if version == "" {
        version = "unknown"
    }

    port := cfg.HealthPort
    if port <= 0 || port > 65535 {
        port = defaultHealthPort
    }

    return &readyz.Config{
        DeploymentMode: mode,
        HealthPort:     port,
        DrainDelay:     drain,
        Version:        version,
    }
}
```

Como o `newReadyzConfig` do manager, mas com o campo extra `HealthPort` clampado.

### Conexões

- **Importa de:** `pkg/bootstrap/readyz/`.
- **Construído por:** `bootstrap/config.go` (em `InitWorker`).
- **Roda sob:** `commons.Launcher` (paralelo ao consumer).

---

## 8. `internal/bootstrap/readyz_adapters.go` — NEW

**Responsabilidade:** ponte entre as deps do worker e o pacote readyz. Espelho funcional do manager, com diferenças.

### Estruturas-chave

```go
type workerReadyzDeps struct {
    cfg            *Config
    mongoClient    *libMongo.Client
    rabbitAdapter  pkgRabbitmq.Adapter
    s3Client       *s3.Client
    mtRedisClient  *redis.Client
    tmClient       readyz.TMClient
    tmMongoManager *tmmongo.Manager
    tmRabbitMgr    *tmrabbitmq.Manager

    closers []func() error
}

func newWorkerReadyzDepsST(cfg *Config, mongoClient *libMongo.Client,
                           storage portStorage.Repository,
                           consumer *workerRabbitAdapters.ConsumerRoutes) *workerReadyzDeps

func newWorkerReadyzDepsMT(cfg *Config, mongoClient *libMongo.Client,
                           storage portStorage.Repository,
                           tmClient *tmclient.Client, tmMongo *tmmongo.Manager,
                           tmRabbit *tmrabbitmq.Manager) *workerReadyzDeps

func (d *workerReadyzDeps) close()
func newReadyzMTRedis(cfg *Config) *redis.Client
type workerRabbitMQAdapterProbe struct { ... }
type s3HeadBucketShim struct { client *s3.Client }
func buildWorkerReadyzCheckers(deps *workerReadyzDeps) []readyz.DependencyChecker
func buildWorkerTenantHandler(readyzCfg *readyz.Config, deps *workerReadyzDeps) fiber.Handler
```

### Walkthrough das diferenças vs manager

#### 8.1. `workerReadyzDeps` — bundle de deps

Por que existe um struct dedicado, em vez de passar tudo separado? Porque `NewHealthServer` recebe um único `deps *workerReadyzDeps`. Mais legível que 8 parâmetros.

`closers []func() error` — slice de funções de cleanup. Cada uma libera um recurso owned pelo readyz wiring (notavelmente o `mtRedisClient` em MT mode). O método `close()` invoca todos.

#### 8.2. Duas factories — `newWorkerReadyzDepsST` e `newWorkerReadyzDepsMT`

```go
func newWorkerReadyzDepsST(
    cfg *Config,
    mongoClient *libMongo.Client,
    storage portStorage.Repository,
    consumer *workerRabbitAdapters.ConsumerRoutes,
) *workerReadyzDeps {
    deps := &workerReadyzDeps{cfg: cfg, mongoClient: mongoClient}

    if consumer != nil {
        deps.rabbitAdapter = consumer.Adapter()  // ← usa o getter que adicionamos
    }

    if s3Repo, ok := storage.(*pkgStorage.S3Repository); ok {
        deps.s3Client = s3Repo.Client()  // ← usa o getter no S3Repository
    }

    return deps
}
```

Type assertion `storage.(*pkgStorage.S3Repository)`. O `portStorage.Repository` é interface — mas só sabemos extrair `*s3.Client` da implementação concreta. Se algum dia houver um `BlobStorageRepository`, esse assert simplesmente falha e `s3Client` fica nil — checker S3 não é registrado. Comportamento safe.

```go
func newWorkerReadyzDepsMT(
    cfg *Config,
    mongoClient *libMongo.Client,
    storage portStorage.Repository,
    tmClient *tmclient.Client,
    tmMongo *tmmongo.Manager,
    tmRabbit *tmrabbitmq.Manager,
) *workerReadyzDeps {
    deps := &workerReadyzDeps{
        cfg:            cfg,
        mongoClient:    mongoClient,
        tmClient:       tmClient,
        tmMongoManager: tmMongo,
        tmRabbitMgr:    tmRabbit,
    }

    if s3Repo, ok := storage.(*pkgStorage.S3Repository); ok {
        deps.s3Client = s3Repo.Client()
    }

    rdb := newReadyzMTRedis(cfg)
    if rdb != nil {
        deps.mtRedisClient = rdb
        deps.closers = append(deps.closers, func() error { return rdb.Close() })
    }

    return deps
}
```

A diferença chave: cria seu próprio `*redis.Client` para multi-tenant Redis (`newReadyzMTRedis`). Por quê? Porque o cliente Redis usado pelo event listener (em `initMultiTenantStack`) é fechado pelo `mtCleanup` — se compartilhasse, o readyz tentaria probar um cliente fechado durante shutdown. Cliente próprio = lifecycle independente. E o `closer` é registrado para liberação durante shutdown.

#### 8.3. `s3HeadBucketShim`

```go
type s3HeadBucketShim struct {
    client *s3.Client
}

func (s *s3HeadBucketShim) HeadBucket(ctx context.Context, params *s3.HeadBucketInput,
                                       optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
    if s == nil || s.client == nil {
        return nil, errS3ClientNil
    }
    return s.client.HeadBucket(ctx, params, optFns...)
}
```

Adapter que faz `*s3.Client` satisfazer a interface `readyz.S3HeadBucketAPI`. **Mas espera** — `*s3.Client` já tem método `HeadBucket` com a assinatura certa. Por que precisa de wrapper?

Resposta: nil-safety. O `*s3.Client` sem wrapper, se for nil, panica em `client.HeadBucket(...)`. O shim adiciona o `if s == nil || s.client == nil` antes. Em código de produção isso quase nunca acontece, mas em teste seams é importante.

#### 8.4. `buildWorkerReadyzCheckers` — diferenças vs manager

```go
checkers := make([]readyz.DependencyChecker, 0, 6)

// MongoDB — igual ao manager
if deps.cfg.MultiTenantEnabled { ... NA ... }
else if deps.mongoClient != nil { ... real ... }

// RabbitMQ — igual ao manager
if deps.cfg.MultiTenantEnabled { ... NA ... }
else if deps.rabbitAdapter != nil { ... real ... }

// Multi-tenant Redis — só worker
if deps.mtRedisClient != nil {
    client := deps.mtRedisClient
    checkers = append(checkers, readyz.NewRedisClientCheckerFromFn(
        "multi_tenant_redis",
        func(ctx context.Context) error { return client.Ping(ctx).Err() },
        readyz.ComposeRedisURL(...),
    ))
}

// S3 — só worker
if deps.s3Client != nil {
    checkers = append(checkers, readyz.NewS3BucketChecker(
        &s3HeadBucketShim{client: deps.s3Client},
        deps.cfg.ObjectStorageBucket,
        deps.cfg.ObjectStorageEndpoint,
    ))
}

// Tenant Manager — igual ao manager
if deps.cfg.MultiTenantEnabled {
    checkers = append(checkers, readyz.NewTenantManagerClientChecker(...))
}
```

Ordem dos blocos: MongoDB, RabbitMQ, Redis, S3, Tenant Manager. Note que **não há** checker para "redis" (schema cache) — isso é só do manager.

Note também que o nome do dep Redis é `"multi_tenant_redis"`, não `"redis"`. Dashboards conseguem distinguir entre o Redis do manager (cache) e o Redis do worker (event discovery) pelo nome.

#### 8.5. `buildWorkerTenantHandler`

```go
func buildWorkerTenantHandler(readyzCfg *readyz.Config, deps *workerReadyzDeps) fiber.Handler {
    if deps == nil || deps.cfg == nil || !deps.cfg.MultiTenantEnabled ||
        deps.tmClient == nil || deps.tmMongoManager == nil || deps.tmRabbitMgr == nil {
        return readyz.NewDisabledTenantHandler()
    }
    ...
}
```

Defesa em profundidade idêntica ao manager — seis condições para ativar.

### Conexões

- **Importa de:** `pkg/bootstrap/readyz/`, `pkg/storage/`, `pkg/rabbitmq/`.
- **Consumido por:** `bootstrap/health_server.go`, `bootstrap/config.go`.

---

## 9. `internal/bootstrap/consumer.go` — MODIFIED

**Responsabilidade:** o consumer RabbitMQ que processa jobs. Agora também coordena drain.

### O que foi adicionado

#### 9.1. Campo `drainDelay` no `MultiQueueConsumer`

```go
type MultiQueueConsumer struct {
    consumerRoutes *rabbitmq.ConsumerRoutes
    mtConsumer     MultiTenantConsumerInterface
    UseCase        *services.UseCase
    logger         libLog.Logger
    queueName      string
    mongoManager   *tmmongo.Manager
    initErr        error
    drainDelay     time.Duration  // ← novo
}
```

Mesma duração do server.go do manager. Configurada no `NewMultiQueueConsumer*` com `defaultDrain(cfg.ReadyzDrainDelaySec)`.

#### 9.2. Goroutine de drain em `Run`

```go
go func() {
    <-sigs

    readyz.SetDraining(true)

    if mq.logger != nil {
        mq.logger.Log(baseCtx, libLog.LevelInfo,
            "received shutdown signal; readyz draining flag set, sleeping drain grace period")
    }

    if mq.drainDelay > 0 {
        select {
        case <-time.After(mq.drainDelay):
        case <-baseCtx.Done():
        }
    }

    if mq.logger != nil {
        mq.logger.Log(baseCtx, libLog.LevelInfo,
            "drain grace period elapsed; cancelling consumer context")
    }

    cancel()
}()
```

Sequência:

1. Espera SIGTERM.
2. **`readyz.SetDraining(true)` — antes de qualquer outra coisa.** Isso é o que muda em relação à versão antiga.
3. Sleep `drainDelay`.
4. `cancel()` — agora os consumers param.

**Por que essa ordem é vital?** Imagina o cenário inverso (cancel primeiro, depois drain flag):

- `cancel()` chega no consumer → `ConsumerLoop` retorna → não puxa mais mensagens.
- Mensagem que estava sendo processada ainda em flight.
- Service faz nack para mensagens novas, mas k8s ainda manda tráfego porque `/readyz=200`.
- Backlog de unack messages cresce.
- Eventually, alerta dispara.

Na ordem certa:

- Drain flag flipa → `/readyz=503`.
- Em ~2s o k8s tira o pod do Service.
- Sleep mais alguns segundos para garantir que tudo já foi removido.
- Cancel → consumer para de puxar mensagens. Toda mensagem em flight termina ack.

#### 9.3. Tudo o resto

Não mudou nada na lógica de processamento de mensagens (`handlerGenerateReport`, `handlerGenerateReportDelivery`, `extractTenantIDFromHeaders`, `resolveTenantMongo`, `isPermanentTenantError`). Esse arquivo é grande, mas o delta de readyz é **só a goroutine de drain**.

### Conexões

- **Importa:** `pkg/bootstrap/readyz/` (apenas para SetDraining).
- **Construído por:** `bootstrap/config.go`.

---

## 10. `internal/bootstrap/service.go` — MODIFIED

**Responsabilidade:** orquestra o lifecycle do worker — Launcher, hooks de shutdown.

### O que foi adicionado

#### 10.1. `runLauncher` reescrito

```go
var runLauncher = func(logger libLog.Logger, consumer *MultiQueueConsumer, healthServer *HealthServer) {
    opts := []commons.LauncherOption{
        commons.WithLogger(logger),
        commons.RunApp("RabbitMQ Consumer", consumer),
    }

    if healthServer != nil {
        opts = append(opts, commons.RunApp("Health Server", healthServer))
    }

    commons.NewLauncher(opts...).Run()
}
```

Antes: rodava só o consumer. Agora: roda consumer **e** healthServer sob o mesmo Launcher. **Mesmo Launcher = mesmo SIGTERM**. Quando SIGTERM chega, ambos param coordenadamente.

#### 10.2. Novos campos em `Service`

```go
type Service struct {
    *MultiQueueConsumer
    libLog.Logger
    licenseShutdown licenseTerminator
    mtCleanup       func()
    healthServer    *HealthServer  // ← novo
    readyzCloser    func()         // ← novo
}
```

`healthServer` é o micro-server. `readyzCloser` é a função `(*workerReadyzDeps).close`.

#### 10.3. Shutdown sequence em `Run`

```go
func (app *Service) Run() {
    runLauncher(app.Logger, app.MultiQueueConsumer, app.healthServer)

    app.Log(context.Background(), libLog.LevelInfo, "Starting graceful shutdown...")

    if app.mtCleanup != nil {
        app.Log(...)
        app.mtCleanup()
        app.Log(...)
    }

    if app.readyzCloser != nil {
        app.readyzCloser()
    }

    if app.licenseShutdown != nil {
        app.licenseShutdown.Terminate("Consumers are done.")
    }

    app.Log(context.Background(), libLog.LevelInfo, "Graceful shutdown complete")
}
```

Sequência ordenada de cleanup: MT (event listener Redis) → readyz (cliente Redis dedicado) → license. Ordem importa porque cada um pode depender de algo que o anterior libera. License por último porque ele aciona um terminate externo.

### Conexões

- **Construído por:** `bootstrap/config.go`.
- **Roda:** `MultiQueueConsumer` + `HealthServer` em paralelo.

---

## 11. `internal/bootstrap/retry_guard.go` — MODIFIED

**Responsabilidade:** classifica erros do handler como retryable ou não.

### O que mudou

**Apenas formatação.** O `gofmt` realinhou os comentários inline da slice `permanentPatterns` em `isPermanentErrorByPattern`. Nenhum delta de comportamento.

### Por que esse arquivo aparece nos commits

Provavelmente um `make lint --fix` que rodou junto com a entrega do readyz. Não é parte da feature — é cleanup oportunístico.

---

## 12. `.env.example` (worker) — MODIFIED

**Responsabilidade:** documentar variáveis de ambiente para dev local.

### O que foi adicionado

```dotenv
# /readyz / health configuration
DEPLOYMENT_MODE=local
HEALTH_PORT=4007
READYZ_DRAIN_DELAY_SEC=12
```

Defaults safe para dev:

- `local` desliga TLS enforcement.
- `4007` é a porta padrão fora do conflito.
- `12s` cobre kube-proxy sync.

Esse arquivo é a "documentação executável" — devs podem fazer `cp .env.example .env` e ter algo que funciona.

---

# Mapa de dependências entre arquivos

Para fixar o aprendizado, aqui um diagrama do quem-importa-quem dentro desses dois componentes:

```
                                       ┌────────────────────────┐
                                       │ pkg/bootstrap/readyz/* │
                                       │ (lib canônica)         │
                                       └────────┬───────────────┘
                                                │
                ┌───────────────────────────────┼───────────────────────────────┐
                ▼                               ▼                               ▼
   ┌────────────────────────┐       ┌────────────────────────┐       ┌────────────────────────┐
   │ Manager                │       │ Worker                 │       │ pkg/storage/s3.go      │
   │                        │       │                        │       │ pkg/rabbitmq/*         │
   │ adapters/http/in/      │       │ internal/adapters/     │       │  (modificados:         │
   │   routes.go            │       │   rabbitmq/...         │       │   getters expostos)    │
   │   ↑                    │       │     consumer.rabbitmq  │       └────────────────────────┘
   │   │ injects handlers   │       │       (Adapter())      │
   │   │                    │       │                        │
   │ bootstrap/             │       │ internal/bootstrap/    │
   │   config.go            │       │   config.go            │
   │   ├─ buildSaaSTLSConfig│       │   ├─ buildSaaSTLSConfig│
   │   ├─ newReadyzConfig   │       │   ├─ newWorkerReadyzCfg│
   │   ├─ runManagerSelfP.  │       │   ├─ runWorkerSelfP.   │
   │   └─ assembleService   │       │   └─ InitWorker        │
   │       ↑                │       │       ↑                │
   │       │ uses           │       │       │ uses           │
   │ bootstrap/             │       │ internal/bootstrap/    │
   │   readyz_adapters.go   │       │   readyz_adapters.go   │
   │   ├─ rabbitMQAdapterPr.│       │   ├─ workerReadyzDeps  │
   │   ├─ buildManagerCh.   │       │   ├─ buildWorkerCh.    │
   │   └─ buildManagerTen.  │       │   └─ buildWorkerTen.   │
   │                        │       │                        │
   │ bootstrap/             │       │ internal/bootstrap/    │
   │   server.go            │       │   health_server.go     │
   │   └─ drainLoop         │       │     (micro-server)     │
   │                        │       │                        │
   │                        │       │ internal/bootstrap/    │
   │                        │       │   consumer.go          │
   │                        │       │     (drain coord)      │
   │                        │       │                        │
   │                        │       │ internal/bootstrap/    │
   │                        │       │   service.go           │
   │                        │       │     (Launcher 2 apps)  │
   └────────────────────────┘       └────────────────────────┘
```

Direção das setas:

- `routes.go` recebe os handlers (não constrói).
- `config.go` é o orquestrador — chama tudo.
- `readyz_adapters.go` traduz tipos do projeto para tipos do readyz.
- No worker, `health_server.go` consome o que `readyz_adapters.go` produz.
- `consumer.go` (worker) e `server.go` (manager) coordenam o drain — flipam o flag de `pkg/bootstrap/readyz/state.go`.

---

# Roteiro sugerido de leitura para fixação

Se você está estudando este código pela primeira vez, leia nessa ordem:

1. **`adapters/http/in/routes.go` (manager)** — entenda como handlers são montados.
2. **`bootstrap/readyz_adapters.go` (manager)** — entenda como deps são adaptadas.
3. **`bootstrap/config.go` (manager)** — entenda a orquestração completa.
4. **`bootstrap/server.go` (manager)** — entenda o drain coordinator.
5. **`internal/bootstrap/health_server.go` (worker)** — entenda a variante micro-server.
6. **`internal/bootstrap/consumer.go` (worker)** — entenda como o consumer participa do drain.

---

# Decisões importantes de design — em uma página

| Decisão | Local | Por quê |
|---|---|---|
| Handlers recebidos como parâmetro em `NewRoutes` | `routes.go` | Inversão de dependência — adapter HTTP não sabe construir readyz. |
| `mongoClient` exposto em `managerRepositories` | `bootstrap/config.go` | Reuso do cliente já criado, sem segunda conexão. |
| `readyzRedisClient` separado do schema cache | `bootstrap/config.go`, `readyz_adapters.go` | Schema cache tem fallback memória — probaria sempre verde. |
| `validateSaaSTLSFn` antes de qualquer `init*` | `bootstrap/config.go` | Boot deve falhar antes de abrir conexões plaintext. |
| Drain flag flipa antes de listener fechar | `server.go`, `consumer.go` | `/readyz=503` precisa chegar ao kube-proxy antes do listener morrer. |
| Variáveis de pacote como seams de teste | múltiplos | Permite override sem usar interfaces. |
| `NewDisabledTenantHandler` em vez de não montar a rota | `routes.go`, `health_server.go` | Operador distingue "MT off" de "rota faltando". |
| Worker tem self-probe próprio antes de consumir | `internal/bootstrap/config.go` | Evita processar mensagens em estado degradado. |
| `*HealthServer` rodando sob mesmo `Launcher` do consumer | `internal/bootstrap/service.go` | SIGTERM unificado coordena ambos. |
| Defesa nil em getters (`Adapter()`, `Client()`) | `consumer.rabbitmq.go`, `s3.go` | Permite paths de teste sem mocks completos. |
| Drain delay clampado para 1s mínimo (não 0) | `server.go`, `health_server.go` | Drain instantâneo derrubaria conexões. |

Cada uma dessas decisões resolve um problema operacional concreto. Quando você ler o código, procure pelo "porquê" em vez de só ler o "como" — assim você aprende padrões transferíveis para outras services.
