# TRD: Crypto Security Enhancement

**Feature:** crypto-security-enhancement
**Gate:** 2 - Technical Requirements Document
**Date:** 2026-01-23
**Status:** Draft

---

## Metadata

```yaml
feature: crypto-security-enhancement
gate: 2
deployment:
  model: Cloud/On-Premise (Hybrid)
tech_stack:
  primary: Go
  standards_loaded:
    - golang.md
    - devops.md
    - sre.md
project_technologies:
  - category: Key Derivation
    prd_requirement: Secure key derivation
    choice: HKDF-SHA256 (golang.org/x/crypto/hkdf)
    rationale: NIST recommended, standard library extension
  - category: Message Authentication
    prd_requirement: Message integrity verification
    choice: HMAC-SHA256 (existing HMACSigner)
    rationale: Already implemented, industry standard
```

---

## 1. Architecture Overview

### 1.1 Architecture Style

**Style:** Hexagonal Architecture with CQRS (existing pattern)

A feature modifica a camada de infraestrutura criptográfica, adicionando derivação de chaves e estendendo os pontos de assinatura/verificação existentes.

### 1.2 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          KEY DERIVATION LAYER                           │
│                                                                         │
│    ┌──────────────────────────────────────────────────────────────┐    │
│    │                     MASTER_KEY (APP_ENC_KEY)                  │    │
│    │                       (existing, unchanged)                   │    │
│    └──────────────────────────────┬───────────────────────────────┘    │
│                                   │                                     │
│                          HKDF Derivation                                │
│                                   │                                     │
│    ┌──────────────┬───────────────┼───────────────┬──────────────┐     │
│    │              │               │               │              │     │
│    ▼              ▼               ▼               ▼              ▼     │
│ ┌──────┐    ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────┐    │
│ │CRED  │    │INT_HMAC  │   │EXT_HMAC  │   │DOC_HMAC  │   │(future)│    │
│ │KEY   │    │KEY       │   │KEY       │   │KEY       │   │        │    │
│ │      │    │          │   │          │   │          │   │        │    │
│ │cred  │    │int-hmac  │   │ext-hmac  │   │doc-hmac  │   │        │    │
│ │-v1   │    │-v1       │   │-v1       │   │-v1       │   │        │    │
│ └──┬───┘    └────┬─────┘   └────┬─────┘   └────┬─────┘   └────────┘    │
│    │             │              │              │                        │
└────┼─────────────┼──────────────┼──────────────┼────────────────────────┘
     │             │              │              │
     ▼             ▼              ▼              ▼
┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│AES-GCM  │  │RabbitMQ  │  │RabbitMQ  │  │Document  │
│Encrypt  │  │Manager↔  │  │Worker→   │  │HMAC      │
│(creds)  │  │Worker    │  │Apps      │  │          │
└─────────┘  └──────────┘  └──────────┘  └──────────┘
```

### 1.3 Component Context

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         MANAGER COMPONENT                                │
│                                                                         │
│  ┌──────────────────┐    ┌──────────────────┐                          │
│  │  KeyDeriver      │───▶│  RabbitMQ        │                          │
│  │  (derives keys)  │    │  Publisher       │                          │
│  │                  │    │  (INT_HMAC_KEY)  │                          │
│  └──────────────────┘    └──────────────────┘                          │
│           │                                                             │
│           ▼                                                             │
│  ┌──────────────────┐                                                  │
│  │  AES-GCM         │                                                  │
│  │  (CRED_KEY)      │                                                  │
│  └──────────────────┘                                                  │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                         WORKER COMPONENT                                 │
│                                                                         │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐  │
│  │  KeyDeriver      │───▶│  RabbitMQ        │───▶│  RabbitMQ        │  │
│  │  (derives keys)  │    │  Consumer        │    │  Publisher       │  │
│  │                  │    │  (INT_HMAC_KEY)  │    │  (EXT_HMAC_KEY)  │  │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘  │
│           │                                                             │
│           ├──────────────────────────────────────┐                      │
│           ▼                                      ▼                      │
│  ┌──────────────────┐                   ┌──────────────────┐           │
│  │  AES-GCM         │                   │  Document HMAC   │           │
│  │  (CRED_KEY)      │                   │  (DOC_HMAC_KEY)  │           │
│  └──────────────────┘                   └──────────────────┘           │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                      CONSUMER APPS (Reporter/Matcher)                    │
│                                                                         │
│  ┌──────────────────┐    ┌──────────────────┐                          │
│  │  RabbitMQ        │───▶│  Document        │                          │
│  │  Consumer        │    │  Verifier        │                          │
│  │  (EXT_HMAC_KEY)  │    │  (DOC_HMAC_KEY)  │                          │
│  └──────────────────┘    └──────────────────┘                          │
│                                                                         │
│  Nota: EXT_HMAC_KEY e DOC_HMAC_KEY são a mesma chave derivada          │
│        (info="fetcher-external-v1") para simplificar distribuição       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Component Design

### 2.1 KeyDeriver (New Component)

**Location:** `pkg/crypto/key_deriver.go`

**Purpose:** Derivar chaves específicas por contexto a partir da master key

**Interface:**
```go
type KeyDeriver interface {
    DeriveKey(context string, length int) ([]byte, error)
    GetCredentialKey() []byte
    GetInternalHMACKey() []byte
    GetExternalHMACKey() []byte
    GetDocumentHMACKey() []byte
}
```

**Responsibilities:**
- Receber master key no bootstrap
- Derivar chaves usando HKDF com info parameters únicos
- Cachear chaves derivadas (derivação única no startup)
- Expor chaves para cada contexto

**Key Contexts:**
| Context | Info Parameter | Purpose |
|---------|---------------|---------|
| Credentials | `fetcher-credentials-v1` | AES-GCM para senhas de conexão |
| Internal HMAC | `fetcher-internal-hmac-v1` | Assinatura Manager↔Worker |
| External HMAC | `fetcher-external-v1` | Assinatura Worker→Apps + HMAC de documentos |

**Design Decision:** External HMAC e Document HMAC usam a mesma chave derivada para simplificar distribuição para consumidores (uma chave para verificar tudo).

### 2.2 HMACSigner Enhancement

**Location:** `pkg/crypto/signer.go`

**Enhancements:**
1. Remover `NewHMACSignerFromCryptor` (deprecated)
2. Adicionar `SignReader(io.Reader) (string, error)` para streaming
3. Manter `NewHMACSigner(key, version)` como construtor primário

**Interface (enhanced):**
```go
type Signer interface {
    Sign(payload []byte) string
    SignReader(r io.Reader) (string, error)
    Verify(payload []byte, signature string) bool
    SignatureVersion() string
}
```

### 2.3 Job Domain Model Enhancement

**Location:** `pkg/model/job.go`

**Enhancement:**
- Adicionar campo `ResultHMAC` à entidade Job
- Adicionar campo `ResultHMAC` ao JobResponse

**Data Ownership:** Worker é o owner do HMAC (write), Manager apenas lê

### 2.4 JobResultData Enhancement

**Location:** `components/worker/internal/services/job_notification.go`

**Enhancement:**
- Adicionar campo `HMAC` ao struct de resultado
- HMAC flui automaticamente para notificação RabbitMQ

### 2.5 Bootstrap Configuration Enhancement

**Locations:**
- `components/manager/internal/bootstrap/config.go`
- `components/worker/internal/bootstrap/config.go`

**Enhancement:**
- Criar KeyDeriver com master key
- Criar signers com chaves derivadas apropriadas
- Configurar RabbitMQ adapters com signers corretos

---

## 3. Data Architecture

### 3.1 Key Derivation Flow

```
┌─────────────────┐
│   MASTER_KEY    │  (APP_ENC_KEY - existing)
│   (32 bytes)    │
└────────┬────────┘
         │
         │ HKDF-SHA256
         │ salt: nil (per RFC 5869)
         │
         ├─── info="fetcher-credentials-v1" ───▶ CRED_KEY (32 bytes)
         │
         ├─── info="fetcher-internal-hmac-v1" ─▶ INT_HMAC_KEY (32 bytes)
         │
         └─── info="fetcher-external-v1" ──────▶ EXT_HMAC_KEY (32 bytes)
                                                 (usado para mensagens E documentos)
```

### 3.2 Data Flow - Message Signing

```
┌─────────┐                              ┌─────────┐
│ Manager │                              │ Worker  │
└────┬────┘                              └────┬────┘
     │                                        │
     │ Sign with INT_HMAC_KEY                 │
     ├────────────────────────────────────────▶
     │      (job message)                     │
     │                                        │
     │                                        │ Verify with INT_HMAC_KEY
     │                                        │
     │                                        │ Process job
     │                                        │
     │                                        │ Compute DOC_HMAC with EXT_HMAC_KEY
     │                                        │
     │                                        │ Sign notification with EXT_HMAC_KEY
     │                                        │
     │      ┌─────────────────────────────────┤
     │      │                                 │
     │      ▼                                 │
     │ ┌─────────┐                            │
     │ │ Apps    │◀───────────────────────────┘
     │ │(Reporter│    (notification + DOC_HMAC)
     │ │ Matcher)│
     │ └────┬────┘
     │      │
     │      │ Verify notification with EXT_HMAC_KEY
     │      │ Download document
     │      │ Verify DOC_HMAC with EXT_HMAC_KEY
     │      │
     │      ▼
     │   Process data
```

### 3.3 Data Model Changes

**Job Entity:**
```
Job
├── ... (existing fields)
├── ResultPath: string
├── ResultSizeBytes: int64
├── ResultRowCount: int64
├── ResultFormat: string
└── ResultHMAC: string (NEW)
```

**MongoDB Document:**
```
jobs collection
├── ... (existing fields)
├── result_path: string
├── result_size_bytes: int64
├── result_row_count: int64
├── result_format: string
└── result_hmac: string (NEW)
```

**JobResultData (notification):**
```
JobResultData
├── Path: string
├── SizeBytes: int64
├── RowCount: int64
├── Format: string
└── HMAC: string (NEW)
```

---

## 4. API Design

### 4.1 GET /v1/fetcher/{id} Response Enhancement

**Enhanced Response:**
```json
{
  "id": "uuid",
  "status": "completed",
  "resultPath": "/path/to/file",
  "resultSizeBytes": 1024,
  "resultRowCount": 100,
  "resultFormat": "csv",
  "resultHmac": "a1b2c3d4..." (NEW - 64 hex chars)
}
```

**Behavior:**
- `resultHmac` presente apenas quando `status == completed`
- `resultHmac` omitido para jobs legados (omitempty)

### 4.2 RabbitMQ Message Format

**Current Headers:**
```
x-message-signature: <signature>
t: <timestamp>
signature-version: <version>
```

**Unchanged** - headers mantidos, apenas chave de assinatura é diferente (EXT_HMAC_KEY vs INT_HMAC_KEY)

**Notification Body Enhancement:**
```json
{
  "jobId": "uuid",
  "status": "completed",
  "result": {
    "path": "/path/to/file",
    "sizeBytes": 1024,
    "rowCount": 100,
    "format": "csv",
    "hmac": "a1b2c3d4..." (NEW)
  }
}
```

---

## 5. Security Architecture

### 5.1 Key Hierarchy

| Level | Key | Derivation | Purpose | Who Has Access |
|-------|-----|------------|---------|----------------|
| 0 | MASTER_KEY | N/A (source) | Derivation source | Ops only |
| 1 | CRED_KEY | HKDF(master, "fetcher-credentials-v1") | Encrypt DB passwords | Manager, Worker |
| 1 | INT_HMAC_KEY | HKDF(master, "fetcher-internal-hmac-v1") | Sign Manager↔Worker | Manager, Worker |
| 1 | EXT_HMAC_KEY | HKDF(master, "fetcher-external-v1") | Sign/verify external | Worker, Consumer Apps |

### 5.2 Key Distribution

**Internal Keys (CRED, INT_HMAC):**
- Derivados de MASTER_KEY (APP_ENC_KEY)
- Nunca saem do Fetcher (Manager/Worker)
- Não precisam de distribuição externa

**External Key (EXT_HMAC):**
- Derivado de MASTER_KEY
- **Precisa ser distribuído para consumidores**
- Opções:
  1. **Derivação local:** Consumidor recebe MASTER_KEY e deriva localmente (não recomendado - expõe master)
  2. **Chave pré-derivada:** Ops calcula EXT_HMAC_KEY e distribui separadamente (recomendado)
  3. **Variável dedicada:** Nova env var `FETCHER_VERIFICATION_KEY` com EXT_HMAC_KEY pré-derivada

**Recomendação:** Opção 2/3 - Ops deriva a chave uma vez e configura em cada consumidor via `FETCHER_VERIFICATION_KEY`

### 5.3 Threat Model

| Threat | Mitigation |
|--------|------------|
| Master key compromise | Rotação: nova master + redeployment |
| External key compromise | Não compromete credenciais (segregação) |
| Replay attack | Timestamp validation (5 min tolerance) |
| Man-in-the-middle | HMAC verification of all messages |
| Document tampering | Document HMAC verification |

### 5.4 Security Controls

- **Fail-fast:** Services fail to start if keys missing/invalid
- **No logging:** Keys never logged
- **Constant-time:** HMAC verification uses constant-time comparison
- **Key versioning:** Info parameter includes version for future rotation

---

## 6. Integration Patterns

### 6.1 Manager Bootstrap

```
1. Load MASTER_KEY from APP_ENC_KEY
2. Create KeyDeriver
3. Derive CRED_KEY → Create AESGCMService
4. Derive INT_HMAC_KEY → Create HMACSigner for internal
5. Configure RabbitMQ publisher with internal signer
```

### 6.2 Worker Bootstrap

```
1. Load MASTER_KEY from APP_ENC_KEY
2. Create KeyDeriver
3. Derive CRED_KEY → Create AESGCMService
4. Derive INT_HMAC_KEY → Create HMACSigner for internal consumer
5. Derive EXT_HMAC_KEY → Create HMACSigner for external publisher + document HMAC
6. Configure RabbitMQ consumer with internal signer (verify)
7. Configure RabbitMQ publisher with external signer (sign)
8. Configure document HMAC with external signer
```

### 6.3 Consumer App Verification

```
1. Load FETCHER_VERIFICATION_KEY (pre-derived EXT_HMAC_KEY)
2. Create HMACSigner for verification
3. On message receive:
   a. Extract signature, timestamp from headers
   b. Validate timestamp within 5 minutes
   c. Rebuild signature payload
   d. Verify signature with constant-time comparison
   e. If valid, process message
4. On document download:
   a. Download document from SeaweedFS
   b. Compute HMAC of content
   c. Compare with resultHmac from message/API
   d. If match, process document
```

---

## 7. Configuration

### 7.1 Environment Variables

**Fetcher (Manager/Worker) - Unchanged:**
| Variable | Required | Description |
|----------|----------|-------------|
| `APP_ENC_KEY` | Yes | Master key (Base64, 32 bytes) |
| `APP_ENC_KEY_VERSION` | Yes | Key version for rotation |

**Consumer Apps - New:**
| Variable | Required | Description |
|----------|----------|-------------|
| `FETCHER_VERIFICATION_KEY` | For verification | Pre-derived EXT_HMAC_KEY (Base64, 32 bytes) |

### 7.2 Key Derivation Script (for Ops)

Documentation will include a script/command to derive EXT_HMAC_KEY from MASTER_KEY for distribution to consumers.

---

## 8. Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| HKDF derivation | < 1ms per key | Called once at startup |
| HMAC computation | < 100ms for 100MB | SHA256 throughput ~500MB/s |
| Message signing | < 1ms | Small payload |
| Message verification | < 1ms | Small payload |
| API latency impact | < 5ms | Single field addition |

---

## 9. Observability

### 9.1 Logging

| Event | Level | Fields |
|-------|-------|--------|
| Keys derived | INFO | key_contexts (list), duration_ms |
| Message signed | DEBUG | message_type, timestamp |
| Message verification failed | WARN | reason, timestamp_diff |
| Document HMAC computed | INFO | job_id, duration_ms, file_size |

### 9.2 Metrics (Future)

| Metric | Type | Description |
|--------|------|-------------|
| `message_signature_verification_total` | Counter | Total verifications |
| `message_signature_verification_failed_total` | Counter | Failed verifications |
| `document_hmac_computation_seconds` | Histogram | HMAC computation time |

---

## 10. Architecture Decision Records (ADRs)

### ADR-001: HKDF for Key Derivation

**Context:** Need to derive multiple keys from single master key.

**Options:**
1. Independent random keys for each context
2. HKDF derivation from master key
3. Simple hash-based derivation (not recommended)

**Decision:** HKDF-SHA256 derivation

**Rationale:**
- NIST recommended (SP 800-56C)
- Single master key reduces key management complexity
- Derived keys are cryptographically independent
- Version in info parameter enables rotation

**Consequences:**
- Requires golang.org/x/crypto/hkdf
- Ops needs tool to derive consumer key

---

### ADR-002: Unified External Key

**Context:** Need to distribute verification key(s) to consumer apps.

**Options:**
1. Separate keys for message verification and document HMAC
2. Single key for all external verification

**Decision:** Single unified external key (EXT_HMAC_KEY)

**Rationale:**
- Simplifies distribution (one key per consumer)
- Both use cases are "external verification"
- Reduces configuration complexity
- Security equivalent (same trust boundary)

**Consequences:**
- Consumers use same key for message and document verification
- If key is compromised, both are affected (acceptable - same trust boundary)

---

### ADR-003: Pre-derived Key Distribution

**Context:** How to distribute verification key to consumers.

**Options:**
1. Share master key with consumers (they derive locally)
2. Pre-derive external key, share only that
3. Use KMS/Vault (out of scope)

**Decision:** Pre-derived key distribution

**Rationale:**
- Master key never leaves Fetcher boundary
- Consumers only get verification capability
- Simpler than KMS integration
- Works for on-premise deployments

**Consequences:**
- Ops derives key once, distributes to each consumer
- Documentation needed for derivation process
- Key rotation requires redistribution

---

### ADR-004: Backward Compatibility via omitempty

**Context:** Existing jobs don't have HMAC. How to handle API responses?

**Decision:** Use `omitempty` for HMAC fields

**Rationale:**
- No breaking change for consumers
- Clear distinction: absent = legacy, present = verified
- Consistent with existing optional fields pattern

**Consequences:**
- Consumers must handle optional field
- No migration needed for existing data

---

### ADR-005: Internal vs External Signing Keys

**Context:** Should Manager↔Worker and Worker→Apps use the same signing key?

**Decision:** Separate keys (INT_HMAC_KEY and EXT_HMAC_KEY)

**Rationale:**
- Different trust boundaries
- Consumer apps should not be able to forge Manager→Worker messages
- Compromised consumer key doesn't affect internal communication
- Principle of least privilege

**Consequences:**
- Two signers in Worker
- Consumer cannot impersonate Manager

---

## 11. File Changes Summary

| File | Type | Change |
|------|------|--------|
| `pkg/crypto/key_deriver.go` | New | HKDF key derivation |
| `pkg/crypto/signer.go` | Enhance | Add SignReader, deprecate FromCryptor |
| `pkg/model/job.go` | Enhance | Add ResultHMAC |
| `pkg/mongodb/job/job.go` | Enhance | Add result_hmac |
| `pkg/mongodb/job/job.mongodb.go` | Enhance | Update mappers |
| `components/manager/internal/bootstrap/config.go` | Enhance | Use KeyDeriver |
| `components/worker/internal/bootstrap/config.go` | Enhance | Use KeyDeriver, dual signers |
| `components/worker/internal/services/job_notification.go` | Enhance | Add HMAC to result |
| `components/worker/internal/services/extract-data.go` | Enhance | Compute document HMAC |
| `internal/services/command/update_job.go` | Enhance | Accept ResultHMAC |
| `.env.example` (all) | Enhance | Document key derivation |

---

## Gate 2 Validation Checklist

- [x] All PRD features mapped to components
- [x] Component boundaries clear
- [x] Interfaces technology-agnostic (where possible)
- [x] Data ownership explicit
- [x] Quality attributes achievable
- [x] Security architecture comprehensive
- [x] ADRs document key decisions

**Status:** ✅ PASS - Ready for Gate 3 (Task Breakdown)
