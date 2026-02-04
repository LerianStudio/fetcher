# PRD: Crypto Security Enhancement

**Feature:** crypto-security-enhancement
**Gate:** 1 - Product Requirements Document
**Date:** 2026-01-23
**Status:** Draft

---

## 1. Executive Summary

Aprimorar a segurança criptográfica do Fetcher através da segregação de chaves por contexto, extensão da verificação de integridade para comunicações Worker → Aplicações, e adição de HMAC aos documentos gerados. Isso permitirá que produtos consumidores (Reporter, Matcher) verifiquem a autenticidade e integridade dos dados recebidos do Fetcher.

---

## 2. Problem Statement

### 2.1 Current Situation

O Fetcher possui criptografia para credenciais de conexão e assinatura de mensagens entre Manager e Worker. No entanto, existem gaps que afetam a segurança e a capacidade de verificação por parte dos consumidores.

### 2.2 Pain Points

| Pain Point | Impact | Evidence |
|------------|--------|----------|
| Produtos consumidores não podem verificar autenticidade das mensagens | Consumidores confiam cegamente nos dados recebidos | Reporter/Matcher não têm acesso a chaves de verificação |
| Documentos extraídos não têm prova de integridade | Impossível detectar adulteração de documentos | Requisito de compliance e auditoria |
| Chave criptográfica compartilhada para múltiplos propósitos | Risco de segurança: comprometimento de uma função afeta outra | Análise de código identificou reutilização de chave |
| Falta de documentação para integração segura | Desenvolvedores não sabem como verificar dados | Sem guia de verificação para consumidores |

### 2.3 User Impact

- **Produtos Lerian (Reporter, Matcher):** Não conseguem verificar se os dados recebidos são autênticos e íntegros
- **Clientes Enterprise:** Requisitos de compliance exigem cadeia de custódia verificável
- **Equipe de Segurança:** Arquitetura atual não segue best practices de key management

---

## 3. User Personas

### 3.1 Produto Consumidor (Reporter/Matcher)

**Demographics:**
- Sistema automatizado que consome dados do Fetcher
- Processa documentos para geração de relatórios ou reconciliação

**Goals:**
- Verificar que mensagens recebidas são autênticas (vieram do Fetcher)
- Verificar que documentos não foram adulterados
- Rejeitar dados suspeitos antes do processamento

**Frustrations:**
- Não tem acesso a mecanismo de verificação
- Confia cegamente nos dados recebidos
- Não pode provar origem dos dados para auditoria

### 3.2 Engenheiro de Integração

**Demographics:**
- Desenvolvedor que integra sistemas com o Fetcher
- Implementa pipelines de dados automatizados

**Goals:**
- Implementar verificação de integridade no pipeline
- Documentar cadeia de custódia para compliance
- Setup simples e bem documentado

**Frustrations:**
- Sem documentação de como verificar dados
- Sem chave de verificação disponível
- Processo de setup não definido

### 3.3 Administrador de Segurança

**Demographics:**
- Responsável pela segurança da infraestrutura
- Gerencia chaves e secrets

**Goals:**
- Garantir segregação adequada de chaves
- Minimizar superfície de ataque
- Facilitar rotação de chaves

**Frustrations:**
- Chaves compartilhadas entre funções diferentes
- Dificuldade em auditar uso de chaves

---

## 4. User Stories

### US-001: Segregação de Chaves Criptográficas

**Como** administrador de segurança
**Eu quero** que cada função criptográfica use sua própria chave
**Para que** o comprometimento de uma chave não afete outras funções

**Acceptance Criteria:**
1. Chave de criptografia de credenciais é separada da chave de assinatura
2. Chave de assinatura interna (Manager↔Worker) é separada da externa (Worker→Apps)
3. Chave de HMAC de documentos é dedicada para esse propósito
4. Todas as chaves são derivadas de forma segura

**Success Metrics:**
- 100% das funções criptográficas usam chaves dedicadas
- Zero compartilhamento de chaves entre contextos diferentes

---

### US-002: Verificação de Mensagens pelos Consumidores

**Como** produto consumidor (Reporter/Matcher)
**Eu quero** verificar a autenticidade das mensagens recebidas do Worker
**Para que** eu possa rejeitar mensagens falsificadas ou adulteradas

**Acceptance Criteria:**
1. Mensagens do Worker para apps incluem assinatura verificável
2. Consumidores têm acesso a chave de verificação
3. Verificação inclui proteção contra replay attacks
4. Mensagens inválidas são claramente identificadas

**Success Metrics:**
- 100% das mensagens enviadas incluem assinatura
- Consumidores podem verificar 100% das mensagens
- Taxa de falsos positivos < 0.01%

---

### US-003: HMAC de Documentos Gerados

**Como** produto consumidor
**Eu quero** verificar a integridade dos documentos baixados
**Para que** eu tenha certeza de que o documento não foi alterado

**Acceptance Criteria:**
1. Todo documento gerado tem HMAC calculado
2. HMAC disponível na consulta do job via API
3. HMAC incluído na mensagem de notificação
4. Documentação explica como verificar o HMAC

**Success Metrics:**
- 100% dos jobs completed têm HMAC
- 100% das notificações de sucesso incluem HMAC
- Verificação bem-sucedida em 100% dos testes

---

### US-004: Distribuição de Chave de Verificação

**Como** engenheiro de integração
**Eu quero** obter a chave de verificação de forma segura
**Para que** eu possa configurar meu sistema para verificar dados do Fetcher

**Acceptance Criteria:**
1. Chave de verificação é diferente da chave de assinatura (se aplicável)
2. Processo de obtenção da chave é documentado
3. Chave pode ser configurada via variável de ambiente
4. Guia de integração disponível

**Success Metrics:**
- Setup de verificação em < 30 minutos seguindo documentação
- 100% dos consumidores autorizados podem obter chave

---

### US-005: Proteção contra Replay Attacks

**Como** produto consumidor
**Eu quero** que mensagens antigas sejam rejeitadas automaticamente
**Para que** atacantes não possam reenviar mensagens capturadas

**Acceptance Criteria:**
1. Mensagens incluem timestamp
2. Mensagens com timestamp muito antigo são rejeitadas
3. Janela de tolerância é configurável (padrão: 5 minutos)
4. Log de mensagens rejeitadas por timestamp

**Success Metrics:**
- 100% das mensagens com timestamp > 5 minutos são rejeitadas
- Zero mensagens replay aceitas em testes

---

### US-006: Backward Compatibility

**Como** administrador de sistema
**Eu quero** que a atualização não quebre sistemas existentes
**Para que** a migração seja suave e sem downtime

**Acceptance Criteria:**
1. Sistemas que não verificam assinatura continuam funcionando
2. Jobs antigos (sem HMAC) são retornados normalmente
3. Migração gradual é possível
4. Documentação de migração disponível

**Success Metrics:**
- Zero breaking changes para consumidores existentes
- Migração completa em < 1 sprint após release

---

## 5. Feature Requirements

### 5.1 Functional Requirements

| ID | Requirement | Priority | User Story |
|----|-------------|----------|------------|
| FR-001 | Segregar chaves criptográficas por contexto | Must Have | US-001 |
| FR-002 | Derivar chaves de forma segura | Must Have | US-001 |
| FR-003 | Assinar mensagens Worker → Apps | Must Have | US-002 |
| FR-004 | Permitir verificação de assinatura por consumidores | Must Have | US-002 |
| FR-005 | Calcular HMAC de documentos gerados | Must Have | US-003 |
| FR-006 | Expor HMAC na API de consulta de job | Must Have | US-003 |
| FR-007 | Incluir HMAC na notificação RabbitMQ | Must Have | US-003 |
| FR-008 | Disponibilizar chave de verificação para consumidores | Must Have | US-004 |
| FR-009 | Incluir timestamp em mensagens para replay protection | Must Have | US-005 |
| FR-010 | Manter backward compatibility | Must Have | US-006 |

### 5.2 Non-Functional Requirements

| ID | Requirement | Target | Rationale |
|----|-------------|--------|-----------|
| NFR-001 | Performance de derivação | < 10ms no startup | Não impactar tempo de inicialização |
| NFR-002 | Performance de HMAC | < 100ms para 100MB | Não impactar tempo de processamento |
| NFR-003 | Latência de API | Aumento < 5ms | Manter SLA existente |
| NFR-004 | Segurança | Chaves 256-bit | Padrão criptográfico moderno |
| NFR-005 | Disponibilidade | Zero downtime na migração | Continuidade de negócio |

---

## 6. Scope

### 6.1 In Scope

- Segregação de chaves criptográficas por contexto (credenciais, HMAC interno, HMAC externo, HMAC de documento)
- Extensão da assinatura HMAC para mensagens Worker → Aplicações consumidoras
- Cálculo e armazenamento de HMAC para documentos gerados
- Exposição do HMAC na API e notificações
- Disponibilização de chave de verificação para consumidores
- Documentação de integração para verificação

### 6.2 Out of Scope

- Rotação automática de chaves (manual via redeploy)
- Interface gráfica para gerenciamento de chaves
- Integração com Key Management Systems externos (Vault, KMS)
- Criptografia end-to-end (já existe para documentos)
- Verificação automática pelo Fetcher (responsabilidade do consumidor)
- Nonce-based replay protection (timestamp é suficiente para o caso de uso)

---

## 7. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Key Segregation | 100% funções com chave dedicada | Code audit |
| HMAC Coverage | 100% jobs completed | Query MongoDB |
| Message Signing | 100% notificações assinadas | Integration tests |
| Verification Success | 100% consumidores podem verificar | Consumer tests |
| Backward Compatibility | 0 breaking changes | Regression tests |
| Documentation | Setup em < 30 min | User testing |

---

## 8. Dependencies

| Dependency | Type | Status |
|------------|------|--------|
| HMACSigner existente | Internal | Disponível |
| golang.org/x/crypto/hkdf | External | Disponível (stdlib extension) |
| RabbitMQ signing pattern | Internal | Disponível |
| MongoDB | External | Operacional |
| SeaweedFS | External | Operacional |

---

## 9. Risks and Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Consumidores não implementam verificação | Medium | Low | Documentação clara, exemplos de código |
| Chave de verificação vaza | Low | Medium | Processo de rotação documentado |
| Breaking change não identificado | Low | High | Testes de regressão extensivos |
| Complexidade de setup para consumidores | Medium | Medium | Automação e exemplos prontos |

---

## 10. Security Requirements (Business Level)

| Requirement | Description |
|-------------|-------------|
| Autenticação de origem | Consumidores podem verificar que dados vieram do Fetcher |
| Integridade de dados | Consumidores podem verificar que dados não foram alterados |
| Segregação de privilégios | Cada contexto criptográfico usa chave dedicada |
| Proteção contra replay | Mensagens antigas são rejeitadas automaticamente |
| Auditabilidade | Chaves e verificações podem ser rastreadas |

---

## Gate 1 Pass Criteria

- [x] Problem articulated clearly
- [x] User impact quantified/qualified
- [x] Users specifically identified (Reporter, Matcher, Engenheiros, Admins)
- [x] Features address core problem (segregação, verificação, HMAC)
- [x] Success metrics measurable
- [x] Scope explicitly bounded
- [x] No technical implementation details
- [x] Security requirements at business level

**Status:** ✅ PASS - Ready for Gate 2 (TRD)
