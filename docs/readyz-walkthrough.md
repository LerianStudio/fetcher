# Walkthrough da entrega `/readyz` no fetcher

> Documento didático cobrindo cada bloco novo dos commits `dcd5fc2`, `2d38baf` e `f7d1754`.

## 1. Por que essa entrega existe

Imagina que você publica um pod na frente de um Service do Kubernetes. Sem um sinal honesto de "estou pronto", o `kube-proxy` começa a roteár tráfego pra você antes do MongoDB conectar, ou mantém você no pool depois que você já recebeu SIGTERM. Resultado: requisições erram em prod.

**`/readyz` é o sinal honesto.** Ele responde a três perguntas:

1. **Estou de pé e funcional agora?** (se cair, devolve `503` e o pod sai do Service)
2. **Cada dependência minha está saudável?** (Mongo, Rabbit, Redis, S3, Tenant Manager)
3. **Estou drenando?** (se SIGTERM já chegou, devolve `503` por um tempo antes de fechar listeners)

A entrega segue um contrato chamado **ring:dev-readyz** — uma especificação interna da Lerian que padroniza o que toda service da frota expõe. Isso permite que dashboards, alertas e probes do Kubernetes funcionem **idênticos** em qualquer service.

A arquitetura final tem três camadas:

```
┌───────────────────────────────────────────────────────────┐
│  Documentação (docs/readyz-guide.md, preview HTML)        │
└───────────────────────────────────────────────────────────┘
                          │
┌───────────────────────────────────────────────────────────┐
│  Wiring nos componentes (manager + worker)                │
│   - constrói os checkers a partir do Config               │
│   - injeta no router/health-server                        │
└───────────────────────────────────────────────────────────┘
                          │
┌───────────────────────────────────────────────────────────┐
│  Pacote canônico: pkg/bootstrap/readyz/                   │
│   - tipos, handler, checkers, métricas, TLS, draining     │
│   - sem dependência das camadas acima → reutilizável      │
└───────────────────────────────────────────────────────────┘
```

A regra de ouro: **o pacote `pkg/bootstrap/readyz/` é puro e fica no fundo do grafo de dependências**. Manager e worker importam dele; ele não importa de ninguém de cima. É o que permite a mesma lib servir reporter, matcher e qualquer outro serviço amanhã.

---

## 2. O pacote canônico — `pkg/bootstrap/readyz/`

Esse pacote é a "biblioteca padrão" do `/readyz`. Vou agrupar por responsabilidade, não na ordem alfabética dos arquivos.

### 2.1 Tipos da resposta — `types.go`

**O que faz:** define o JSON do `/readyz`.

```
DependencyCheck {
  status         // "up" | "down" | "degraded" | "skipped" | "n/a"
  latency_ms
  tls            // *bool — nil oculta o campo
  error          // só em down/degraded
  reason         // só em skipped/n/a
  breaker_state  // só quando há circuit breaker
}

ReadyzResponse {
  status         // "healthy" iff todo check estiver em {up, skipped, n/a}
  checks         // map dep → DependencyCheck
  version
  deployment_mode
  tenant_id      // só em /readyz/tenant/:id
}
```

**Por que entrou:** o vocabulário de cinco status (não dois) é proposital. **"skipped"** existe para falar "esse dep está desligado por feature flag — não é falha". **"n/a"** existe para multi-tenant: o Mongo da tenant não cabe no `/readyz` global; precisamos sinalizar que existe e apontar pra `/readyz/tenant/:id`. Sem esse vocabulário, dashboards confundem "desabilitado" com "quebrado".

**Como ajuda na entrega:** é o "esquema do banco de dados" da resposta. Tudo o que vem depois (handler, checkers) só preenche essa estrutura.

**Aprendizado:** repare em `TLS *bool`. Pointer-bool é o truque clássico em Go pra ter três estados em JSON: `nil` (campo some), `*false`, `*true`. Esse padrão aparece muito em APIs públicas.

---

### 2.2 Estado do processo — `state.go`

**O que faz:** dois `atomic.Bool` globais — `drainingState` e `selfProbeOK`.

**Por que entrou:** drain e startup são **condições de processo**, não de uma instância. O manager tem um servidor Fiber, o worker tem um micro-servidor de saúde — ambos precisam ler a mesma flag. Variável de pacote com `atomic.Bool` é a leitura mais barata possível, segura pra ser consultada em todo request de `/readyz`.

**Como ajuda na entrega:** é o canal que conecta o handler `SIGTERM` (em `consumer.go`/`server.go`) ao handler HTTP do `/readyz`. Sem esse estado compartilhado, drain não funciona.

**Aprendizado:** quando você precisa de uma flag binária consultada em hot path, `atomic.Bool` é o padrão. Mutex seria desperdício, e `chan` complica a leitura.

---

### 2.3 Configuração — `config.go`

**O que faz:** `LoadConfig()` lê 4 envs (`DEPLOYMENT_MODE`, `HEALTH_PORT`, `READYZ_DRAIN_DELAY_SEC`, `OTEL_RESOURCE_SERVICE_VERSION`) e devolve um `*Config` independente.

**Por que entrou:** o pacote precisa ler env, **mas** sem importar o struct `Config` do manager nem do worker (causaria ciclo). Solução: um Config próprio, isolado.

**Como ajuda:** todo handler precisa saber em que `deployment_mode` está e qual versão emitir. Centralizar a leitura aqui garante que manager e worker tratam os mesmos valores da mesma forma.

**Aprendizado de design:** ao invés de erros em config inválida, ele faz **fallback pra defaults seguros**. Um `/readyz` que não sobe é um pod cego. Operadores precisam **ver** a configuração ruim na resposta, não num crashloop.

---

### 2.4 O coração — `handler.go`

**O que faz:**

1. Ao chegar request, lê `IsDraining()` — se `true`, devolve 503 com check sintético `"draining"` e nem roda os probes.
2. Senão, dispara **uma goroutine por checker em paralelo**, cada uma sob `context.WithTimeout(PerDepTimeout(name))`.
3. Coleta resultados num channel buferizado, fecha com `WaitGroup.Wait()`.
4. Agrega: `healthy` se todo check estiver em `{up, skipped, n/a}`; senão `unhealthy` + 503.

**Por que entrou assim:**

- **Sem cache.** Cachear `/readyz` cria janelas cegas onde o pod já degradou mas o k8s continua mandando tráfego. O custo é rodar todos os probes a cada request (~1s no pior caso, geralmente <10ms).
- **Paralelo, não serial.** Se serializar, um Mongo lento de 2s + Redis lento de 1s = 3s; em paralelo é 2s. O probe do K8s tem orçamento finito.
- **Per-dep timeout fixo (não configurável).** Mongo = 2s, Redis = 1s, S3 = 2s, etc. Manter fixo garante que dashboards de latência funcionam idênticos em toda a frota — operador não precisa decorar valores diferentes por service.
- **`runWithDeadline` envolve cada checker.** Se um checker mal-comportado ignorar o `ctx`, o handler ainda devolve resultado em tempo hábil substituindo por um "down sintético".

**Como ajuda:** é o motor. Tudo o resto serve a ele.

**Aprendizado:** o padrão "channel buffered + WaitGroup" é o jeito canônico em Go de fazer fan-out sem leak de goroutine. `chan` com tamanho fixo + `wg.Wait()` antes do `close()` garante que toda goroutine termina antes de você ler.

---

### 2.5 Interface dos probers — `checker.go`

**O que faz:** define a interface `DependencyChecker` (`Name`, `Check(ctx)`) e a tabela de timeouts `PerDepTimeout`. Inclui também um `StubChecker` que sempre devolve "skipped" (útil pra wiring incremental).

**Por que entrou:** sem essa interface, cada novo dep teria que mexer no handler. Com ela, registrar um novo checker é uma linha.

**Como ajuda:** é o ponto de extensão. Toda a arquitetura "Mongo, Rabbit, Redis, S3..." só funciona porque eles compartilham essa interface.

**Aprendizado:** repare como `PerDepTimeout` faz fallback por **prefixo** (`upstream_*`, `rabbitmq_*`). É o padrão "convenção sobre configuração": se você nomeia certo, o orçamento é automático.

---

### 2.6 Os probers concretos — `checker_*.go`

Cada arquivo implementa `DependencyChecker` para um sistema externo:

| Arquivo | Dep | Como prova que está vivo |
|---|---|---|
| `checker_mongo.go` | `mongodb` | `Ping()` na conexão |
| `checker_redis.go` | `redis` (e `multi_tenant_redis`) | `PING` |
| `checker_rabbitmq.go` | `rabbitmq` | inspeciona `IsHealthy()` do adapter; consulta `State()` do circuit breaker |
| `checker_s3.go` | `s3` | `HeadBucket` |
| `checker_tmclient.go` | `tenant_manager` | `GetActiveTenantsByService` (probe-and-catch porque o breaker é opaco) |
| `checker_na.go` | qualquer | sempre devolve `n/a` — usado para deps tenant-scoped no `/readyz` global |

**Por que cada um existe separado:** cada sistema tem nuances. Rabbit tem circuit breaker observável; o tenant manager tem breaker não observável. Mongo tem TLS via URI scheme; Redis tem via `rediss://`; S3 tem via endpoint vazio = HTTPS. Encapsular cada particularidade num arquivo dedicado mantém o handler genérico.

**Como ajudam:** cada checker é a tradução "eu estou ok pra esse dep" → `DependencyCheck{Status: ...}`. Sem eles, o handler só sabe agregar — não sabe **o quê** agregar.

**Aprendizado importante: por que não abrir um channel novo no probe do Rabbit?** Olha o comentário em `checker_rabbitmq.go`: o adapter já tem um *channel watcher* que reconecta. Se o probe abrisse um channel a cada request de `/readyz`, em pouco tempo estouraria o limite de channels do servidor RabbitMQ. **Ler estado existente é melhor que provocar I/O.** Esse tipo de decisão é fácil de errar.

**Detalhe crítico do `checker_na.go`:** parece bobo — devolve "n/a" e pronto. Mas a regra do contrato é: **nunca omita silenciosamente um dep do response**. Se `mongodb` é tenant-scoped em multi-tenant, ele tem que aparecer no `/readyz` global como `n/a` com `reason="multi-tenant: see /readyz/tenant/:id"`. Operador precisa **ver** que o dep existe. Esse é o tipo de detalhe que evita meses de "achei que não tinha Mongo nesse service".

---

### 2.7 Métricas Prometheus — `metrics.go`

**O que faz:** registra três métricas e o handler `/metrics`:

```
readyz_check_duration_ms (HistogramVec)  — labels dep+status
readyz_check_status      (CounterVec)    — labels dep+status
selfprobe_result         (GaugeVec)      — label dep
```

E expõe via `promhttp.Handler()`.

**Por que entrou:** `/readyz` te diz "agora", métrica te diz "ao longo do tempo". Sem o histograma, você não vê deterioração lenta de latência; sem o counter, não consegue alertar em "rate de degraded subindo".

**Como ajuda:** transforma a entrega de "endpoint que responde" em "observabilidade". Os nomes e labels são parte do contrato — dashboards Grafana esperam exatamente esses três.

**Aprendizado pontual:** o handler chama `emitCheckStatus` **incondicionalmente** mesmo quando draining. Por quê? `rate()` em Prometheus precisa de série contínua. Se você omite labels, queries quebram com "no data" justamente quando você quer alertar. Métrica deve ser **sempre emitida**.

---

### 2.8 Registro de rotas — `registration.go`

**O que faz:** o helper `Register(app, h)` mounta `/health`, `/readyz`, `/readyz/tenant/:id`, `/metrics` num app Fiber. Também expõe `HealthHandler()` e `StubTenantHandler()`.

**Por que entrou:** padroniza a montagem. O worker usa `Register` direto; o manager monta as rotas no router próprio dele (que tem mais coisas). Mesmo wiring, dois consumidores.

**Como ajuda:** garante que toda service registra no mesmo conjunto de paths e na mesma ordem (antes do auth middleware — probes não usam token).

**`HealthHandler` versus `/readyz`:** dois endpoints, propósitos diferentes:
- `/health` → liveness (você está vivo? se não, k8s mata o pod). Lê só `selfProbeOK`. **Sem I/O**, super barato.
- `/readyz` → readiness (você pode receber tráfego? se não, k8s tira do Service mas não mata).

Isso é importante: se o Mongo cair temporariamente, você quer **sair do Service** (readiness=false), não **morrer** (liveness=true). Esse split evita restart loops sob falha de dep.

---

### 2.9 Multi-tenant — `tenant_handler.go` e `tenant_probers.go`

**O que fazem:** o endpoint `/readyz/tenant/:id` valida que a tenant existe (chama Tenant Manager) e roda probes só pra ela — Mongo da tenant, vhost AMQP da tenant. Os probers vêm em `tenant_probers.go` (`TenantMongoChecker`, `TenantRabbitMQChecker`).

**Por que entraram:** em multi-tenant, cada tenant tem seu próprio Mongo db e seu próprio vhost AMQP. Probar tudo no `/readyz` global seria caro (cardinalidade altíssima) e errado (uma tenant fora do ar não significa o serviço está fora). Solução: endpoint separado, parametrizado pelo `id`.

**Como ajudam:** completam o quadro multi-tenant. Sem isso, o sistema só tem visibilidade do nível plataforma; não consegue diagnosticar "o tenant X não consegue produzir relatório".

**Detalhe esperto:** `runTenantChecks` reusa as mesmas métricas do `/readyz` global (`readyz_check_*`) — **sem** label de `tenant_id`. Isso mantém a cardinalidade do Prometheus controlada. Tenant info aparece no body da resposta JSON, não na métrica.

---

### 2.10 Segurança e robustez — `tls_detection.go`, `tls_enforcement.go`, `sanitize.go`

**`tls_detection.go`:** funções puras `detect{Mongo,Redis,AMQP,S3,HTTPUpstream}TLS(uri) → (bool, error)`. Olham só **scheme/query**, nunca abrem socket. Por quê? **Detectar postura configurada é diferente de validar negociação ao vivo.** Quem precisa do segundo é o probe; quem precisa do primeiro é dashboard ("esse dep está com TLS habilitado?").

**`tls_enforcement.go`:** `ValidateSaaSTLS(SaaSTLSConfig)` — chamada uma vez no boot, **antes** de qualquer conexão. Em `DEPLOYMENT_MODE=saas`, falha o boot se algum dep estiver sem TLS. Em `byoc`/`local`, não faz nada.

Por que isso entrou: regulamentação financeira. Você não pode subir um pod SaaS com Mongo plaintext. Antes era possível porque cada `init*` tinha (ou não) sua própria checagem espalhada. Agora a checagem é centralizada — uma única função, uma única vez, antes do primeiro `Dial`.

**`sanitize.go`:** redação de credenciais em strings de erro. `mongodb://user:pass@host` → `mongodb://***@host`. Tem cap de 512 bytes pra evitar erro gigante vazar a connection string inteira pro Loki.

Por que existe: erros de driver costumam embutir a connection string. Sem sanitização, qualquer falha de Mongo enviaria a senha pro Grafana.

**Como esses três ajudam na entrega:** transformam `/readyz` de "ferramenta de operador" em "ferramenta de operador que não causa incidente sozinha". TLS-on no boot evita 60-day pentests achando ferida. Sanitize evita post-mortem por vazamento.

**Aprendizado importante (reforço de contrato):** o comentário em `tls_enforcement.go` diz que `ValidateSaaSTLS` é o **único lugar** permitido pra `DEPLOYMENT_MODE == "saas"`. Sempre que vir uma proibição assim, é porque alguém já cometeu o anti-pattern de espalhar a checagem em vários `init*` — e isso quebra a próxima vez que você adiciona um novo dep e esquece um lugar.

---

### 2.11 Self-probe de startup — `selfprobe.go`

**O que faz:** `RunSelfProbe(ctx, checkers, logger)` roda todos os checkers uma vez no boot e flipa `selfProbeOK` baseado no agregado. Falha **não** crasha o pod — só loga. `/health` continua devolvendo 503 até o flag virar.

**Por que entrou:** cenário clássico — pod sobe, MongoDB ainda não está reachable. Sem self-probe, `/health` devolveria 200 imediatamente e o k8s declararia o pod ready. Aí chega request, conecta no Mongo, dá timeout. Self-probe vira o "estou cego até provar contrário".

**Por que não crasha:** zero-panic policy. Pod precisa **ficar vivo** pra `/health` servir 503 e o kubelet coletar logs. Crashar antes do log ser escrito é tiro no pé operacional.

**Como ajuda:** completa o ciclo de boot. Sem ele, liveness é otimista demais.

---

## 3. Wiring no Manager — `components/manager/internal/bootstrap/`

O manager já tinha boot infra; precisava só plugar o readyz. Dois arquivos novos:

### 3.1 `readyz_adapters.go` — a "tomada"

**O que faz:** três coisas:

1. **`rabbitMQAdapterProbe`** — adapta `pkgRabbitmq.Adapter` (do projeto) à interface `readyz.RabbitMQAdapterProbe`. Tradução simples de enum (`CircuitOpen` → `BreakerOpen` etc.) sem importar nada de mais alto.

2. **`newReadyzRedisClient`** — cria um `*redis.Client` **dedicado** pro readyz. Por quê não reusar o do schema cache? Porque o schema cache embrulha redis em fallback memória — probar através dele sempre dá "ok" mesmo com Redis down. Cliente dedicado força I/O real.

3. **`buildManagerReadyzCheckers`** — a função-fábrica. Lê o `Config`, decide os checkers a registrar. Em multi-tenant, troca Mongo/Rabbit por NAChecker; em single-tenant, usa probers reais. Adiciona `tenant_manager` só em MT mode.

**Por que entrou:** isolar a lógica de wiring fora do `config.go` gigante. `config.go` cuida de "construir as deps"; `readyz_adapters.go` cuida de "expor essas deps pro readyz".

**Como ajuda:** sem isso, `assembleService` em `config.go` viraria 1500 linhas. Modular: cada arquivo tem uma razão de mudar.

### 3.2 `routes.go` (modificado)

Foi estendido pra aceitar `readyzHandler`, `readyzTenantHandler`, `metricsHandler` como parâmetros e mountá-los **antes** de auth middleware.

**Por que mexeu aí:** probes não autenticam. Se mountasse depois do `auth.Authorize(...)`, o k8s precisaria de service token — não funciona.

### 3.3 `server.go` (modificado)

Ganhou `drainLoop`: intercepta `SIGTERM` próprio (em vez de deixar pro `lib-commons`), flipa `readyz.SetDraining(true)`, dorme `drainDelay`, **só então** sinaliza shutdown.

**Por que mexeu:** o `lib-commons.ServerManager.WithShutdownHook` roda hooks **depois** de fechar listener. Tarde demais — quando o listener fecha, in-flight requests morrem. Precisa flipar drain **antes** de fechar listener pra k8s tirar do Service.

**Aprendizado de operação:** a sequência correta de drain é sempre `flag de saúde → grace → fechar listeners`. Inverter qualquer um desses três causa drop de requests.

---

## 4. Wiring no Worker — `components/worker/internal/bootstrap/`

O worker tem um problema único: ele é um consumidor RabbitMQ, **não** tem servidor HTTP primário. Mas o k8s precisa de readiness probe. Solução: micro-servidor dedicado.

### 4.1 `health_server.go` — o micro-servidor

**O que faz:** monta um `fiber.App` mínimo com `/health`, `/readyz`, `/readyz/tenant/:id`, `/metrics` num `HEALTH_PORT` separado (default 4007). Roda sob o mesmo `commons.Launcher` do consumer, então um único SIGTERM derruba os dois.

**Por que entrou:** sem isso, k8s não tem com quem falar. Probes precisam de uma porta TCP; consumer não tem.

**Como ajuda:** torna o worker observável e operável pelo k8s exatamente como o manager. Mesmas rotas, mesmo contrato.

**Detalhe:** `newWorkerReadyzConfig` faz **clamp** de port (0/negativo/>65535 → 4007). Por quê? Pra um workspace mal configurado não escutar em `:0` (porta aleatória) e quebrar o probe.

### 4.2 `readyz_adapters.go` — gêmeo do manager

Mesma ideia do manager, com diferenças:
- Worker tem **S3** (manager não tem) → registra `S3BucketChecker`.
- Worker não tem schema-cache Redis → o Redis dele é só `multi_tenant_redis`.
- Worker tem `s3HeadBucketShim` adaptando `*s3.Client` à interface do checker.

**Tem duas variantes de construção** — `newWorkerReadyzDepsST` (single-tenant) e `newWorkerReadyzDepsMT` (multi-tenant). MT cria um `*redis.Client` próprio pro probe (mesmo motivo do manager: não compartilhar lifecycle com event listener).

### 4.3 `service.go` (modificado)

`runLauncher` agora aceita `*HealthServer` e roda como segundo `RunApp`. `Service` ganhou `healthServer` e `readyzCloser`. Shutdown invoca o closer pra liberar o redis dedicado.

### 4.4 `consumer.go` (modificado)

A goroutine de SIGTERM agora flipa `readyz.SetDraining(true)` **antes** do sleep de grace, e só então cancela o contexto do consumer. Mesmo padrão do manager: sinal de saúde primeiro, recursos depois.

### 4.5 `consumer.rabbitmq.go` (modificado, do worker adapter)

Ganhou um getter `Adapter()` para expor o `rabbitmq.Adapter` interno. Necessário porque o `readyz_adapters.go` precisa inspecionar o circuit breaker e a flag `IsHealthy()`. Sem o getter, exigiria refletir/expor campos.

---

## 5. Suporte e dependências externas

### 5.1 `pkg/storage/s3.go` (modificado)

Ganhou três getters: `Client()`, `Bucket()`, `Endpoint()`. O motivo é o mesmo do `Adapter()` no rabbit — readyz precisa observar/probar o cliente sem reconstruir um do zero.

### 5.2 `go.mod` / `go.sum` (modificados)

Entraram duas dependências:
- `github.com/prometheus/client_golang` → métricas.
- `go.uber.org/goleak` → detecção de leak de goroutine nos testes.

Por que goleak? Em código com fan-out paralelo (handler do readyz, self-probe), é fácil escrever um bug onde uma goroutine fica pendurada. `goleak` força você a detectar isso em CI. Olha `pkg/bootstrap/readyz/goleak_test.go`.

### 5.3 `.env.example` em manager e worker (modificados)

Documentam os novos envs (`DEPLOYMENT_MODE`, `HEALTH_PORT`, `READYZ_DRAIN_DELAY_SEC`, etc.) com valores default seguros pra dev local.

---

## 6. Documentação operacional

### 6.1 `docs/readyz-guide.md` — o manual do operador

**O que tem:** explica em prosa quando ler `/readyz`, o que cada status significa, como diagnosticar (Mongo down? "down + error" no checker `mongodb`), como configurar o k8s livenessProbe vs readinessProbe.

**Por que entrou:** quem está de plantão às 3 AM precisa abrir um doc e entender o sistema sem mergulhar em código. É o "Stack Overflow específico desse projeto".

### 6.2 `docs/readyz-implementation-preview.html` — preview visual

Página HTML auto-contida que mostra exemplos de respostas JSON pra cada cenário (saudável, draining, multi-tenant, falha de dep). Fica como referência viva — operador olha o HTML pra saber o que esperar antes mesmo de chamar o endpoint em prod.

### 6.3 `docs/ring-dev-readyz/current-cycle.json`

Output do ciclo de implementação do `ring:dev-readyz` skill — quais portões passaram, quais agentes rodaram. É **trilha de auditoria**, não documentação de uso. Útil pra entender por que decisões foram tomadas, mas não pra operar.

---

## 7. Testes — o ponto cego que virou prática

### 7.1 Testes unitários — `*_test.go` ao lado de cada arquivo

Padrão Go: cada `.go` tem seu `_test.go`. O que vale comentar:

- **`handler_test.go`** verifica o invariante do agregado: qualquer `down`/`degraded` flipa pra `unhealthy`.
- **`tls_enforcement_test.go`** roda os 18 cenários da matriz "modo × dep × URL" pra garantir que SaaS exige e BYOC libera.
- **`sanitize_test.go`** tem table-driven com URIs venenosas pra garantir que nenhum padrão escapa.

### 7.2 `goleak_test.go`

Roda `goleak.VerifyNone(t)` ao final de cada teste relevante. Se alguma goroutine ficar viva, falha. Em código com 5+ goroutines paralelas por request, isso é defesa essencial.

### 7.3 `tests/chaos/readyz_chaos_test.go` — o teste mais "do mundo real"

Usa **Toxiproxy** pra injetar latência/perda de pacote/fechamento de conexão entre o pod e os deps. Verifica que `/readyz`:

- Devolve `down` quando dep cai.
- **Volta** pra `up` em ≤2s quando dep recupera (importante: não quer-se sticky failure).
- Mantém `breaker_state="open"` consistente com o circuit breaker.

**Por que esse teste entrou:** unit tests provam lógica. Chaos prova **comportamento sob falha real**. Sem ele, é fácil fazer um `/readyz` que funciona em CI mas trava sob jitter.

---

## 8. Resumo: como tudo conecta

Recapitulando o fluxo completo de uma request `/readyz` no manager em multi-tenant:

```
1. k8s readinessProbe → GET /readyz (sem token, antes do auth middleware)
                             │
2.                          ▼
   handler.Fiber()  ──→ IsDraining()? ─sim─→ 503 + "draining" (skip probes)
                             │ não
                             ▼
3.   handler.Run(ctx)  ──→  fan-out goroutines:
                             ├─ MongoClientChecker.Check()  (single-tenant)
                             │     OU
                             │  NAChecker (multi-tenant: "see /tenant/:id")
                             ├─ RabbitMQAdapterChecker.Check() ou NAChecker
                             ├─ RedisClientChecker.Check() (schema cache)
                             ├─ TenantManagerClientChecker.Check() (só em MT)
                             ▼
4.   aggregateStatus() → "healthy" se todos em {up, skipped, n/a}
                             │
5.                          ▼
   emite métricas (readyz_check_*) → /metrics serve via promhttp
                             │
6.                          ▼
   resposta JSON (200 ou 503)
```

E no SIGTERM:

```
1. SIGTERM chega
2. drainLoop intercepta → readyz.SetDraining(true)
3. /readyz começa a devolver 503 ("draining")
4. k8s tira o pod do Service em ≤2s
5. dorme drainDelay (default 12s = sync window do kube-proxy)
6. fecha listener Fiber, executa hooks de shutdown
7. processo termina
```

Cada arquivo dessa entrega serve um pedaço desse fluxo. Tirar qualquer um quebra uma garantia operacional específica — a que o comentário do código documenta.

---

## Como continuar aprendendo

Ordem sugerida pra leitura mais profunda:

1. `pkg/bootstrap/readyz/types.go` — vocabulário.
2. `pkg/bootstrap/readyz/handler.go` — motor.
3. `pkg/bootstrap/readyz/checker_mongo.go` — um exemplo de prober concreto.
4. `pkg/bootstrap/readyz/tls_enforcement.go` — caso interessante de "uma única função, todo o domínio".
5. `components/manager/internal/bootstrap/readyz_adapters.go` — wiring real.
6. `tests/chaos/readyz_chaos_test.go` — comportamento sob falha.

Pra cada um: leia o código, depois leia o teste correspondente. O teste sempre revela o invariante que o código está tentando preservar.
