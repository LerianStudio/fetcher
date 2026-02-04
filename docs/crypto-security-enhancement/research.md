# Research: Crypto Security Enhancement

**Feature:** crypto-security-enhancement
**Research Mode:** modification
**Date:** 2026-01-23

---

## Executive Summary

A pesquisa identificou **gaps críticos de segurança** na arquitetura criptográfica atual do Fetcher:

1. **Chave compartilhada:** HMAC signer reutiliza a mesma chave de criptografia AES (`APP_ENC_KEY`)
2. **Consumidores sem acesso:** Worker assina mensagens, mas apps consumidoras não têm como verificar
3. **Sem HMAC de documento:** Documentos gerados não têm prova de integridade

A solução recomendada é **segregar chaves por contexto** usando HKDF para derivação, e definir estratégia de distribuição para consumidores.

---

## 1. Estado Atual das Chaves Criptográficas

### 1.1 Inventário de Chaves

| Variável | Componente | Propósito | Encoding | Localização |
|----------|-----------|---------|----------|-------------|
| `APP_ENC_KEY` | Manager, Worker | AES-GCM para senhas de conexão | Base64 (32 bytes) | pkg/crypto/crypto.go |
| `APP_ENC_KEY_VERSION` | Manager, Worker | Versão para rotação | String | pkg/crypto/crypto.go |
| `CRYPTO_ENCRYPT_SECRET_KEY_SEAWEEDFS` | Worker | Criptografia de documentos extraídos | Hex (32 bytes) | extract-data.go:470 |
| `CRYPTO_HASH_SECRET_KEY_SEAWEEDFS` | Worker | Hash para lib-commons crypto | Hex | extract-data.go:475 |
| `CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM` | Worker | Decriptação de dados CRM | Hex | extract_crm_data.go:243 |
| `CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM` | Worker | Hash para filtros CRM | Hex | extract_crm_data.go:131 |

### 1.2 Arquitetura Atual

```
┌─────────────────────────────────────────────────────────────────┐
│                        APP_ENC_KEY                               │
│                    (Base64, 32 bytes)                            │
│                                                                  │
│    ┌──────────────────────┐    ┌──────────────────────┐         │
│    │   AES-GCM Service    │    │    HMAC Signer       │         │
│    │   (credentials)      │◄───│   (REUTILIZA CHAVE)  │ ⚠️ GAP  │
│    │   pkg/crypto/crypto  │    │   pkg/crypto/signer  │         │
│    └──────────────────────┘    └──────────────────────┘         │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                  CRYPTO_*_SEAWEEDFS                              │
│                    (Hex, 32 bytes)                               │
│                                                                  │
│    ┌──────────────────────┐                                     │
│    │   lib-commons Crypto │    Documentos extraídos             │
│    │   (encrypt + hash)   │    são encriptados                  │
│    └──────────────────────┘                                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Gaps de Segurança Identificados

### 2.1 GAP CRÍTICO: Chave Compartilhada (Encryption + HMAC)

**Localização:** `pkg/crypto/signer.go:63-72`

```go
// PROBLEMA: HMAC signer reutiliza chave de criptografia
func NewHMACSignerFromCryptor(cryptor Cryptor) (*HMACSigner, error) {
    aesService, ok := cryptor.(*AESGCMService)
    // ...
    return NewHMACSigner(aesService.key, SignatureVersion)  // ⚠️ MESMA CHAVE
}
```

**Violação:** Princípio "One Key, One Purpose" (OWASP Key Management)

**Risco:** Comprometimento de uma operação (ex: HMAC) pode afetar a outra (ex: AES)

**Recomendação:** Usar chave HMAC dedicada derivada via HKDF

### 2.2 GAP: Consumidores Não Podem Verificar Assinaturas

**Situação Atual:**
- Manager → Worker: Mensagens assinadas ✅
- Worker → Apps: Mensagens assinadas, mas apps não têm chave para verificar ❌

**Localização:** `components/worker/internal/bootstrap/config.go:130-131`

```go
// Worker usa HMAC signer para notificações
messageSigner, err := crypto.NewHMACSignerFromCryptor(cryptoService)
rabbitMQOptions.Signer = messageSigner
```

**Problema:** Apps consumidoras (Reporter, Matcher) não têm `APP_ENC_KEY` e não devem ter (least privilege)

**Recomendação:** Definir estratégia de distribuição de chave de verificação para consumidores

### 2.3 GAP: Documentos Sem HMAC

**Situação Atual:** Documentos são encriptados e armazenados, mas não há HMAC para verificação de integridade

**Problema:** Consumidor não pode verificar se o documento foi alterado após geração

**Recomendação:** Calcular HMAC do documento original, armazenar no Job, expor na API e notificação

---

## 3. Padrões Existentes no Codebase

### 3.1 Assinatura de Mensagens RabbitMQ

**Localização:** `pkg/rabbitmq/rabbitmq.go:678-692`

```go
// Publicação com assinatura
if prmq.options.Signer != nil && prmq.options.EnableMessageSigning {
    timestamp := time.Now().UTC().Unix()
    payload := crypto.BuildSignaturePayload(timestamp, queueMessage)
    signature := prmq.options.Signer.Sign(payload)

    headers[constant.HeaderMessageSignature] = signature
    headers[constant.HeaderSignatureTimestamp] = strconv.FormatInt(timestamp, 10)
    headers[constant.HeaderSignatureVersion] = prmq.options.Signer.SignatureVersion()
}
```

**Verificação:** `pkg/rabbitmq/rabbitmq.go:1114-1202`
- Replay protection: 5 minutos de tolerância
- Constant-time comparison

### 3.2 Headers de Assinatura

**Localização:** `pkg/constant/rabbitmq.go:4-13`

```go
HeaderMessageSignature   = "x-message-signature"
HeaderSignatureTimestamp = "t"
HeaderSignatureVersion   = "signature-version"
```

### 3.3 HMACSigner Existente

**Localização:** `pkg/crypto/signer.go:39-104`

```go
type HMACSigner struct {
    key     []byte
    version string
}

func (s *HMACSigner) Sign(payload []byte) string {
    mac := hmac.New(sha256.New, s.key)
    mac.Write(payload)
    return hex.EncodeToString(mac.Sum(nil))
}

func (s *HMACSigner) Verify(payload []byte, signature string) bool {
    expectedSig := s.Sign(payload)
    return hmac.Equal([]byte(expectedSig), []byte(signature))
}
```

---

## 4. Best Practices Pesquisadas

### 4.1 Princípio: Uma Chave, Um Propósito

**Fonte:** OWASP Key Management Cheat Sheet

> "A single cryptographic key should be used for only one purpose (encryption, authentication, key wrapping, digital signatures). Never use the same key for both encryption and HMAC operations."

### 4.2 HKDF para Derivação de Chaves

**Fonte:** NIST SP 800-56C, Trail of Bits

```
masterKey = secureRandom(32)
encryptionKey = HKDF-Expand(masterKey, info="fetcher-encryption-v1", length=32)
hmacKey = HKDF-Expand(masterKey, info="fetcher-hmac-v1", length=32)
credentialKey = HKDF-Expand(masterKey, info="fetcher-credentials-v1", length=32)
```

**Vantagem:** Uma master key, múltiplas chaves derivadas independentes

### 4.3 HMAC com Replay Protection

**Fonte:** Thomas Rones, Authgear

```
message = {
  payload: <data>,
  timestamp: <ISO8601 UTC>,
  nonce: <UUID v4>,
  signature: HMAC-SHA256(key, canonicalize(payload + timestamp + nonce))
}

Verificação:
1. Check timestamp within acceptable window (+/- 5 minutes)
2. Check nonce not in recent cache
3. Recompute HMAC with constant-time comparison
4. Store nonce in cache
```

### 4.4 Distribuição de Chaves para Consumidores

| Padrão | Prós | Contras | Quando Usar |
|--------|------|---------|-------------|
| **Centralized KMS** (Vault) | Audit trail, rotação automática | Dependência de infra | Produção, compliance |
| **Derived Verification Keys** | Simples, baixa latência | Distribuir para cada consumidor | Trust médio |
| **Shared Secret + Derivation** | Setup mais simples | Todos podem derivar todas as chaves | Desenvolvimento, alta confiança |

---

## 5. Arquitetura Proposta

### 5.1 Hierarquia de Chaves

```
                    +------------------+
                    |   MASTER_KEY     |
                    | (APP_ENC_KEY)    |
                    +--------+---------+
                             |
              HKDF com info parameters únicos
                             |
         +-------------------+-------------------+
         |                   |                   |
+--------v--------+ +--------v--------+ +--------v--------+
| CREDENTIAL_KEY  | | INTERNAL_HMAC   | | EXTERNAL_HMAC   |
| info="cred-v1"  | | info="int-v1"   | | info="ext-v1"   |
| (senhas DB)     | | (Manager↔Worker)| | (Worker→Apps)   |
+-----------------+ +-----------------+ +--------+--------+
                                                 |
                                     Derivada e compartilhada
                                     com apps consumidoras
                                                 |
                               +-----------------+-----------------+
                               |                 |                 |
                           Reporter          Matcher           Outros
```

### 5.2 Segregação por Contexto

| Contexto | Chave Derivada | Quem Usa |
|----------|----------------|----------|
| Criptografia de credenciais | `CREDENTIAL_KEY` | Manager, Worker |
| HMAC interno (Manager↔Worker) | `INTERNAL_HMAC_KEY` | Manager, Worker |
| HMAC externo (Worker→Apps) | `EXTERNAL_HMAC_KEY` | Worker (sign), Apps (verify) |
| HMAC de documentos | `DOCUMENT_HMAC_KEY` | Worker (sign), Apps (verify) |
| Criptografia de documentos | Mantém `CRYPTO_*_SEAWEEDFS` | Worker |

### 5.3 Distribuição para Consumidores

**Opção Recomendada:** Chave de verificação derivada

```
1. Worker deriva EXTERNAL_HMAC_KEY de MASTER_KEY usando info="external-hmac-v1"
2. Apps consumidoras recebem EXTERNAL_HMAC_KEY via:
   - Variável de ambiente (FETCHER_VERIFICATION_KEY)
   - Ou Vault/KMS (produção)
3. Apps usam EXTERNAL_HMAC_KEY apenas para verificar (não assinar)
```

---

## 6. Decisões de Design Propostas

| Decisão | Escolha | Rationale |
|---------|---------|-----------|
| Derivação de chaves | HKDF-SHA256 | Padrão da indústria, suportado em Go |
| Master key | Reutilizar APP_ENC_KEY | Backward compatible, reduz chaves a gerenciar |
| Chaves separadas para | Credentials, Internal HMAC, External HMAC, Document HMAC | One key, one purpose |
| Formato de info | `fetcher-{context}-v{version}` | Versionamento para rotação |
| Distribuição para apps | Variável de ambiente + documentação | Simples, funciona on-premise |
| HMAC de documento | Antes da encriptação | Verifica conteúdo original |

---

## 7. Arquivos a Modificar

| Arquivo | Modificação |
|---------|-------------|
| `pkg/crypto/signer.go` | Adicionar HKDF, remover derivação de AES key |
| `pkg/crypto/crypto.go` | Manter AES-GCM, expor método para HKDF |
| `components/manager/internal/bootstrap/config.go` | Derivar chaves separadas |
| `components/worker/internal/bootstrap/config.go` | Derivar chaves separadas |
| `components/worker/internal/services/extract-data.go` | Calcular HMAC de documento |
| `components/worker/internal/services/job_notification.go` | Incluir HMAC na notificação |
| `pkg/model/job.go` | Adicionar ResultHMAC |
| `pkg/mongodb/job/job.go` | Adicionar result_hmac |
| `.env.example` (todos) | Documentar novas variáveis |

---

## 8. Riscos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|---------------|---------|-----------|
| Breaking change para consumidores | Média | Alto | Backward compatible: aceitar mensagens sem assinatura durante transição |
| Complexidade de distribuição de chaves | Média | Médio | Documentação clara, automação de setup |
| Performance do HKDF | Baixa | Baixo | HKDF é rápido, derivação apenas no startup |
| Erros de implementação HKDF | Média | Alto | Usar biblioteca padrão Go (golang.org/x/crypto/hkdf) |

---

## 9. Referências

### Standards
- OWASP Key Management Cheat Sheet
- NIST SP 800-56C Key Derivation
- RFC 5869 HKDF

### Implementações de Referência
- HashiCorp Vault Transit Secrets Engine
- Netflix Passport (Edge Authentication)
- AWS Signature Version 4

### Artigos
- Trail of Bits: Best Practices for Key Derivation
- Authgear: HMAC API Security
- Thomas Rones: HMAC Timestamp Nonce

---

## Gate 0 Pass Criteria

- [x] Research mode determined: **modification**
- [x] Existing patterns identified: **RabbitMQ signing, HMACSigner, AES-GCM**
- [x] Security gaps identified: **3 gaps críticos**
- [x] Best practices documented: **HKDF, key separation, HMAC replay protection**
- [x] Files to modify identified: **~10 arquivos**

**Status:** ✅ PASS - Pronto para Gate 1 (PRD)
