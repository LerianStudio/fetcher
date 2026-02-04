# Task Breakdown: Crypto Security Enhancement

**Feature:** crypto-security-enhancement
**Gate:** 3 - Task Breakdown
**Date:** 2026-01-23
**Status:** Draft

---

## Executive Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 7 |
| Total Points | 39 |
| Estimated Duration | 2-3 weeks |
| Team Size | 1-2 developers |

---

## Task Overview

| ID | Title | Type | Size | Points | Dependencies |
|----|-------|------|------|--------|--------------|
| T-001 | Implement KeyDeriver with HKDF | Foundation | M | 5 | None |
| T-002 | Enhance HMACSigner (SignReader + Deprecation) | Foundation | S | 3 | None |
| T-003 | Add ResultHMAC to Domain Models | Foundation | S | 3 | None |
| T-004 | Segregate Keys in Manager Bootstrap | Feature | M | 5 | T-001 |
| T-005 | Segregate Keys in Worker + Document HMAC | Feature | L | 13 | T-001, T-002, T-003 |
| T-006 | Consumer Verification Documentation + Tool | Feature | M | 5 | T-001 |
| T-007 | End-to-End Integration Testing | Integration | M | 5 | T-004, T-005 |

---

## Task Dependencies Graph

```
T-001 (KeyDeriver) ─────────┬───▶ T-004 (Manager Bootstrap) ──┐
                            │                                  │
T-002 (HMACSigner) ─────────┼───▶ T-005 (Worker + Doc HMAC) ──┼───▶ T-007 (E2E Tests)
                            │              ▲                   │
T-003 (Domain Models) ──────┴──────────────┘                   │
                                                               │
T-001 (KeyDeriver) ─────────────▶ T-006 (Docs + Tool) ─────────┘
```

---

## Detailed Task Specifications

### T-001: Implement KeyDeriver with HKDF

**Type:** Foundation
**Deliverable:** New KeyDeriver component that derives context-specific keys from master key

#### Scope

**Includes:**
- `pkg/crypto/key_deriver.go` with HKDF implementation
- Support for multiple key contexts (credentials, internal-hmac, external)
- Key caching (derive once at startup)
- Unit tests with test vectors

**Excludes:**
- Integration with bootstrap (T-004, T-005)
- Key rotation logic (out of scope)

#### Success Criteria

**Functional:**
- [ ] DeriveKey(context, length) returns deterministic derived key
- [ ] Same master + context always produces same derived key
- [ ] Different contexts produce cryptographically independent keys
- [ ] GetCredentialKey(), GetInternalHMACKey(), GetExternalHMACKey() work correctly

**Technical:**
- [ ] Uses golang.org/x/crypto/hkdf
- [ ] Keys derived once and cached
- [ ] Info parameter includes version (e.g., "fetcher-credentials-v1")

**Quality:**
- [ ] Unit tests with known test vectors (RFC 5869 vectors)
- [ ] Tests verify key independence (derived keys don't correlate)

#### User/Technical Value

**User Value:** Foundation for secure key segregation across all crypto operations.

**Technical Value:** Enables T-004, T-005, T-006 - all depend on key derivation.

#### Technical Components

**From TRD:**
- KeyDeriver interface and implementation

**Files to Create:**
- `pkg/crypto/key_deriver.go`
- `pkg/crypto/key_deriver_test.go`

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Medium (M) |
| Points | 5 |
| Duration | 2 days |
| Team | Backend |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| HKDF implementation error | High | Low | Use stdlib, verify with RFC test vectors |
| Wrong info parameter format | Medium | Medium | Document format, code review |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| Unit | RFC 5869 test vectors, key independence |
| Unit | All context methods |

#### Definition of Done

- [ ] Code reviewed and approved
- [ ] Unit tests passing with RFC test vectors
- [ ] No lint errors
- [ ] Merged to develop

---

### T-002: Enhance HMACSigner (SignReader + Deprecation)

**Type:** Foundation
**Deliverable:** HMACSigner with streaming support and deprecated legacy constructor

#### Scope

**Includes:**
- `SignReader(io.Reader) (string, error)` method for streaming
- Deprecation annotation on `NewHMACSignerFromCryptor`
- Unit tests for SignReader
- Benchmark for large file performance

**Excludes:**
- Removal of NewHMACSignerFromCryptor (breaking change - future)
- Integration changes

#### Success Criteria

**Functional:**
- [ ] SignReader computes same HMAC as Sign for identical content
- [ ] SignReader handles io.Reader of any size
- [ ] SignReader returns error on io.Reader failure
- [ ] NewHMACSignerFromCryptor marked as deprecated

**Technical:**
- [ ] Memory usage O(1) regardless of input size
- [ ] Backward compatible (existing code still works)

**Quality:**
- [ ] Unit tests cover: empty, small, large input
- [ ] Benchmark shows <100ms for 100MB

#### User/Technical Value

**User Value:** Enables HMAC computation for large documents without memory issues.

**Technical Value:** Foundation for T-005 (document HMAC).

#### Technical Components

**From TRD:**
- HMACSigner enhancement

**Files to Modify:**
- `pkg/crypto/signer.go`
- `pkg/crypto/signer_test.go`

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Small (S) |
| Points | 3 |
| Duration | 1 day |
| Team | Backend |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking existing code | Medium | Low | Deprecate only, don't remove |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| Unit | SignReader equivalence, error handling |
| Benchmark | 100MB in <100ms |

#### Definition of Done

- [ ] Code reviewed and approved
- [ ] Unit tests passing
- [ ] Benchmark meets target
- [ ] Deprecation documented
- [ ] No lint errors
- [ ] Merged to develop

---

### T-003: Add ResultHMAC to Domain Models

**Type:** Foundation
**Deliverable:** Domain models with HMAC field across all layers

#### Scope

**Includes:**
- ResultHMAC in Job entity
- ResultHMAC in JobResponse DTO
- result_hmac in MongoDB model
- Mapper updates
- HMAC in JobResultData (notification)

**Excludes:**
- API handler changes (automatic via response mapping)
- Worker computation (T-005)

#### Success Criteria

**Functional:**
- [ ] Job entity has ResultHMAC string field
- [ ] JobResponse has ResultHmac with `json:"resultHmac,omitempty"`
- [ ] MongoDB model has result_hmac with `bson:"result_hmac,omitempty"`
- [ ] JobResultData has HMAC with `json:"hmac,omitempty"`
- [ ] Mappers transfer HMAC correctly

**Technical:**
- [ ] All existing tests pass
- [ ] omitempty ensures backward compatibility

**Quality:**
- [ ] Swagger docs regenerated

#### User/Technical Value

**User Value:** API and notifications will expose HMAC.

**Technical Value:** Foundation for T-005 and API exposure.

#### Technical Components

**From TRD:**
- Job Domain Model modification
- MongoDB Job Model modification
- JobResultData modification

**Files to Modify:**
- `pkg/model/job.go`
- `pkg/mongodb/job/job.go`
- `pkg/mongodb/job/job.mongodb.go`
- `components/worker/internal/services/job_notification.go`

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Small (S) |
| Points | 3 |
| Duration | 1 day |
| Team | Backend |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Missing mapper field | Medium | Low | Test roundtrip |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| Unit | Mapper roundtrip |
| Regression | Existing tests pass |

#### Definition of Done

- [ ] Code reviewed and approved
- [ ] All existing tests pass
- [ ] Swagger regenerated
- [ ] No lint errors
- [ ] Merged to develop

---

### T-004: Segregate Keys in Manager Bootstrap

**Type:** Feature
**Deliverable:** Manager uses derived keys instead of shared key

#### Scope

**Includes:**
- KeyDeriver initialization in Manager bootstrap
- CRED_KEY for AESGCMService
- INT_HMAC_KEY for RabbitMQ publisher (Manager→Worker)
- Backward compatible (same master key)

**Excludes:**
- Worker changes (T-005)
- New environment variables

#### Success Criteria

**Functional:**
- [ ] Manager starts successfully with KeyDeriver
- [ ] AESGCMService uses CRED_KEY (derived)
- [ ] RabbitMQ publisher uses INT_HMAC_KEY (derived)
- [ ] Existing functionality unchanged (encrypt/decrypt, message signing)

**Technical:**
- [ ] Keys derived at startup, not per-operation
- [ ] Logs show key derivation success

**Quality:**
- [ ] Existing integration tests pass

#### User/Technical Value

**User Value:** Improved security through key segregation.

**Technical Value:** Manager-side of the key segregation story.

#### Technical Components

**From TRD:**
- Manager bootstrap configuration

**Files to Modify:**
- `components/manager/internal/bootstrap/config.go`

#### Dependencies

| Type | Task | Reason |
|------|------|--------|
| Requires | T-001 | KeyDeriver implementation |

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Medium (M) |
| Points | 5 |
| Duration | 2 days |
| Team | Backend |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking Manager↔Worker communication | High | Medium | Test with Worker before merge |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| Unit | Bootstrap initializes correctly |
| Integration | Manager↔Worker still works |

#### Definition of Done

- [ ] Code reviewed and approved
- [ ] Manager starts with derived keys
- [ ] Integration test: Manager→Worker message works
- [ ] No lint errors
- [ ] Merged to develop

---

### T-005: Segregate Keys in Worker + Document HMAC

**Type:** Feature
**Deliverable:** Worker uses derived keys, computes document HMAC, signs external notifications

#### Scope

**Includes:**
- KeyDeriver initialization in Worker bootstrap
- CRED_KEY for AESGCMService
- INT_HMAC_KEY for RabbitMQ consumer verification
- EXT_HMAC_KEY for RabbitMQ publisher (external notifications)
- EXT_HMAC_KEY for document HMAC computation
- Document HMAC calculation in extract-data flow
- HMAC storage in Job document
- HMAC in notification message

**Excludes:**
- Consumer verification (T-006)
- E2E tests (T-007)

#### Success Criteria

**Functional:**
- [ ] Worker starts with KeyDeriver
- [ ] RabbitMQ consumer verifies with INT_HMAC_KEY
- [ ] RabbitMQ publisher signs with EXT_HMAC_KEY
- [ ] Document HMAC computed before encryption
- [ ] HMAC stored in MongoDB job document
- [ ] HMAC included in notification message
- [ ] HMAC visible in GET /v1/fetcher/{id} response

**Technical:**
- [ ] Uses SignReader for streaming HMAC
- [ ] HMAC logged with duration
- [ ] Keys never logged

**Operational:**
- [ ] No new environment variables (uses existing APP_ENC_KEY)

**Quality:**
- [ ] Integration test verifies HMAC presence

#### User/Technical Value

**User Value:** Documents and notifications now have verifiable integrity.

**Technical Value:** Core feature - enables consumer verification.

#### Technical Components

**From TRD:**
- Worker bootstrap
- ExtractData UseCase
- Notification publishing

**Files to Modify:**
- `components/worker/internal/bootstrap/config.go`
- `components/worker/internal/services/extract-data.go`
- `internal/services/command/update_job.go`

#### Dependencies

| Type | Task | Reason |
|------|------|--------|
| Requires | T-001 | KeyDeriver |
| Requires | T-002 | SignReader |
| Requires | T-003 | Domain models with HMAC |

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Large (L) |
| Points | 13 |
| Duration | 5-7 days |
| Team | Backend |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Breaking Worker→Manager verification | High | Medium | Coordinate with T-004 |
| HMAC of wrong content | High | Medium | Clear code comments, tests |
| Performance regression | Medium | Low | Streaming HMAC (SignReader) |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| Unit | Key derivation, HMAC computation |
| Integration | Job has HMAC, notification has HMAC |
| Integration | HMAC matches document content |

#### Definition of Done

- [ ] Code reviewed and approved
- [ ] Worker starts with derived keys
- [ ] Document HMAC computed and stored
- [ ] Notification includes HMAC
- [ ] API returns HMAC
- [ ] Integration tests pass
- [ ] No lint errors
- [ ] Deployed to staging

---

### T-006: Consumer Verification Documentation + Tool

**Type:** Feature
**Deliverable:** Documentation and CLI tool for consumers to verify Fetcher data

#### Scope

**Includes:**
- Documentation: how to obtain verification key
- Documentation: how to verify message signatures
- Documentation: how to verify document HMAC
- CLI tool or script to derive EXT_HMAC_KEY from MASTER_KEY
- Example code snippets (Go)

**Excludes:**
- SDK library for consumers (future)
- Automatic key distribution

#### Success Criteria

**Functional:**
- [ ] CLI tool derives EXT_HMAC_KEY from MASTER_KEY
- [ ] Documentation explains message verification
- [ ] Documentation explains document HMAC verification
- [ ] Example code works correctly

**Technical:**
- [ ] Tool output matches Worker's derived key
- [ ] Examples use constant-time comparison

**Quality:**
- [ ] Documentation reviewed for clarity
- [ ] Setup achievable in <30 minutes

#### User/Technical Value

**User Value:** Consumers can now verify data integrity.

**Technical Value:** Completes the security story - verification is usable.

#### Technical Components

**From TRD:**
- Key distribution strategy
- Consumer verification flow

**Files to Create:**
- `docs/security/verification-guide.md`
- `scripts/derive-verification-key.go` (or shell script)

#### Dependencies

| Type | Task | Reason |
|------|------|--------|
| Requires | T-001 | KeyDeriver to derive key |

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Medium (M) |
| Points | 5 |
| Duration | 2 days |
| Team | Backend + Docs |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Documentation unclear | Medium | Medium | User testing, review |
| Tool produces wrong key | High | Low | Test against Worker |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| Manual | Follow docs, verify setup works |
| Script | Tool output matches Worker's key |

#### Definition of Done

- [ ] Documentation reviewed
- [ ] CLI tool works correctly
- [ ] Example code tested
- [ ] Setup achievable in <30 min
- [ ] Merged to develop

---

### T-007: End-to-End Integration Testing

**Type:** Integration
**Deliverable:** Full flow verified: key segregation → signing → verification → HMAC

#### Scope

**Includes:**
- Test: Manager→Worker uses INT_HMAC_KEY
- Test: Worker→Apps uses EXT_HMAC_KEY
- Test: Document HMAC in API and notification
- Test: Consumer can verify with derived key
- Test: Backward compatibility (legacy jobs work)

**Excludes:**
- Performance testing
- Chaos testing

#### Success Criteria

**Functional:**
- [ ] Manager→Worker communication works with derived keys
- [ ] Worker→Apps notifications are signed
- [ ] Document HMAC matches actual content
- [ ] Consumer verification succeeds with derived key
- [ ] Legacy jobs (no HMAC) work correctly

**Technical:**
- [ ] Tests run in CI
- [ ] Tests use testcontainers

**Quality:**
- [ ] Full coverage of happy path + edge cases

#### User/Technical Value

**User Value:** Confidence that security enhancement works end-to-end.

**Technical Value:** Regression protection.

#### Technical Components

**From TRD:**
- Full integration flow

**Files to Create/Modify:**
- `tests/integration/containers/crypto_test.go`

#### Dependencies

| Type | Task | Reason |
|------|------|--------|
| Requires | T-004 | Manager segregation |
| Requires | T-005 | Worker segregation + HMAC |

#### Effort Estimate

| Attribute | Value |
|-----------|-------|
| Size | Medium (M) |
| Points | 5 |
| Duration | 2 days |
| Team | Backend/QA |

#### Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Flaky tests | Medium | Medium | Appropriate timeouts |
| Complex test setup | Medium | Medium | Clear documentation |

#### Testing Strategy

| Type | Coverage |
|------|----------|
| E2E | Full flow verification |
| Regression | Legacy compatibility |

#### Definition of Done

- [ ] Integration tests passing in CI
- [ ] All scenarios covered
- [ ] No flaky tests
- [ ] Merged to develop

---

## Delivery Sequencing

### Phase 1: Foundation (T-001, T-002, T-003) - Parallel

**Goal:** Establish infrastructure for key segregation

| Task | Team | Notes |
|------|------|-------|
| T-001 | Backend | KeyDeriver (critical path) |
| T-002 | Backend | HMACSigner enhancement |
| T-003 | Backend | Domain model changes |

**Can run in parallel** - no dependencies between them.

### Phase 2: Core Implementation (T-004, T-005, T-006)

**Goal:** Implement key segregation and HMAC

| Task | Team | Notes |
|------|------|-------|
| T-004 | Backend | Manager bootstrap (depends on T-001) |
| T-005 | Backend | Worker + HMAC (depends on T-001, T-002, T-003) |
| T-006 | Backend/Docs | Documentation (depends on T-001) |

**T-004 and T-006 can start as soon as T-001 is done.**
**T-005 needs all foundation tasks.**

### Phase 3: Validation (T-007)

**Goal:** Verify end-to-end functionality

| Task | Team | Notes |
|------|------|-------|
| T-007 | Backend/QA | Integration tests (depends on T-004, T-005) |

---

## Confidence Score

| Factor | Points | Criteria |
|--------|--------|----------|
| Task Decomposition | 27/30 | All tasks appropriately sized |
| Value Clarity | 23/25 | Every task delivers working software |
| Dependency Mapping | 24/25 | All dependencies documented |
| Estimation Quality | 17/20 | Based on similar work |

**Total:** 91/100

**Action:** Autonomous execution recommended

---

## Gate 3 Validation Checklist

- [x] All TRD components have tasks
- [x] All PRD features have tasks
- [x] Each task appropriately sized (no XL+)
- [x] Task boundaries clear
- [x] Every task delivers working software
- [x] User value explicit
- [x] Technical value clear
- [x] Sequence optimizes value
- [x] Success criteria measurable/testable
- [x] Dependencies correctly mapped
- [x] Testing approach defined
- [x] DoD comprehensive
- [x] Risks identified per task
- [x] Mitigations defined

**Status:** ✅ PASS - Small Track Complete

---

## Summary

```
✅ Small Track (4 gates) complete for crypto-security-enhancement

Artifacts created:
- docs/pre-dev/crypto-security-enhancement/research.md (Gate 0)
- docs/pre-dev/crypto-security-enhancement/prd.md (Gate 1)
- docs/pre-dev/crypto-security-enhancement/trd.md (Gate 2)
- docs/pre-dev/crypto-security-enhancement/tasks.md (Gate 3)

Feature scope:
1. Key segregation via HKDF (credentials, internal HMAC, external HMAC)
2. Document HMAC (storage, API, notification)
3. Consumer verification (documentation, tool)

Total: 7 tasks, 39 points, ~2-3 weeks

Next steps:
1. Review artifacts in docs/pre-dev/crypto-security-enhancement/
2. Use /worktree to create isolated workspace
3. Use /write-plan to create implementation plan
4. Execute with /dev-cycle
```
