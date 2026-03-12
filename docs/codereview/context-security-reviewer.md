# Pre-Analysis Context: Security

## Security Scanner Findings (1 issues)


| Severity | Tool | Rule | File | Line | Message |
|----------|------|------|------|------|---------|
| high | gosec | G115 | components/worker/internal/bootstrap/config.go | 186 | integer overflow conversion int -&gt; uint64 |


## Data Flow Analysis


### High Risk Flows (41)

#### flow\_428464f56770de4a: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:252`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `conn, err := h.GetQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_ee91e2370c100f71: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:252`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.TestQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_6d0fa301d63ad5f8: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:252`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `conn, err := h.UpdateCmd.Execute\(ctx, orgID, id, request\)`
**Sanitized:** No

#### flow\_fa86b88b9e5e77e7: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:252`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_ce16f0f4d81cfe83: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:252`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.GetSchemaQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_6d6be168120361c7: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:315`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.TestQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_113ada7c64ad289a: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:315`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `conn, err := h.UpdateCmd.Execute\(ctx, orgID, id, request\)`
**Sanitized:** No

#### flow\_2b05a630f35c1e27: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:315`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_3cf6a2c101656862: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:315`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.GetSchemaQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_431f5d86a2adb3c5: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:380`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `conn, err := h.UpdateCmd.Execute\(ctx, orgID, id, request\)`
**Sanitized:** No

#### flow\_ded5efdf89dc97ce: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:380`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_599419b660b10c1f: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:380`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.GetSchemaQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_4ae7f0447055b21d: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:467`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_70b905a7b20a2837: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:467`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.GetSchemaQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_a267f98c8566a07d: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/connection.go:600`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `resp, err := h.GetSchemaQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_eb658cd8763f9437: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/fetcher.go:190`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `job, err := h.GetJobQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_8942fc094911ed96: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/migration.go:140`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'connectionID' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `connectionID, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `conn, err := h.AssignCmd.Execute\(ctx, orgID, connectionID, productID\)`
**Sanitized:** No

#### flow\_0809741f90181e77: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/product.go:228`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `product, err := h.GetQuery.Execute\(ctx, orgID, id\)`
**Sanitized:** No

#### flow\_87b65b5a70eafde6: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/product.go:228`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `product, err := h.UpdateCmd.Execute\(ctx, orgID, id, request\)`
**Sanitized:** No

#### flow\_bd2772790933490b: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/product.go:228`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_ef868691fbca90ed: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/product.go:292`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `product, err := h.UpdateCmd.Execute\(ctx, orgID, id, request\)`
**Sanitized:** No

#### flow\_cc27c0658b4b899e: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/product.go:292`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_d27ed86b6d47054e: http\_path -> template
**File:** `components/manager/internal/adapters/http/in/product.go:379`
**Risk:** high
**Notes:** Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Source:** `id, err := uuid.Parse\(c.Params\("id"\)\)`
**Sink:** `if err := h.DeleteCmd.Execute\(ctx, orgID, id\); err != nil {`
**Sanitized:** No

#### flow\_86f1f5c08d38144c: http\_body -> template
**File:** `pkg/itestkit/addons/queuekit/assertions.go:251`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(expected, &e\); err != nil {`
**Sink:** `if err := json.Unmarshal\(actual, &a\); err != nil {`
**Sanitized:** No

#### flow\_62349e425f3f5193: http\_body -> template
**File:** `pkg/itestkit/addons/queuekit/assertions.go:251`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(expected, &e\); err != nil {`
**Sink:** `if err := json.Unmarshal\(data, &m\); err != nil {`
**Sanitized:** No

#### flow\_72db8ca2936846be: http\_body -> template
**File:** `pkg/itestkit/addons/queuekit/assertions.go:256`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(actual, &a\); err != nil {`
**Sink:** `if err := json.Unmarshal\(data, &m\); err != nil {`
**Sanitized:** No

#### flow\_cd31b4fafee8776c: http\_body -> template
**File:** `pkg/itestkit/addons/queuekit/matcher.go:153`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(msg.Body, &data\); err != nil {`
**Sink:** `if err := json.Unmarshal\(msg.Body, &data\); err != nil {`
**Sanitized:** No

#### flow\_c4917cb7e4910667: http\_body -> template
**File:** `pkg/itestkit/addons/queuekit/matcher.go:153`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(msg.Body, &data\); err != nil {`
**Sink:** `if err := json.Unmarshal\(msg.Body, &data\); err != nil {`
**Sanitized:** No

#### flow\_2c6cf36889d9b6b9: http\_body -> template
**File:** `pkg/itestkit/addons/queuekit/matcher.go:167`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(msg.Body, &data\); err != nil {`
**Sink:** `if err := json.Unmarshal\(msg.Body, &data\); err != nil {`
**Sanitized:** No

#### flow\_486fb4281b2592ec: http\_body -> template
**File:** `pkg/mysql/datasource.mysql.go:375`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sanitized:** No

#### flow\_d6fa159a5dd1806d: http\_body -> template
**File:** `pkg/mysql/datasource.mysql.go:375`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_8b922a7fd9b0c924: http\_body -> template
**File:** `pkg/mysql/datasource.mysql.go:381`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_b1bd46a6068057b3: http\_body -> template
**File:** `pkg/oracle/datasource.oracle.go:629`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sanitized:** No

#### flow\_e72fbf3a0d800ad5: http\_body -> template
**File:** `pkg/oracle/datasource.oracle.go:629`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_cd4308d050770680: http\_body -> template
**File:** `pkg/oracle/datasource.oracle.go:635`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_689628cc7959a4bf: http\_body -> template
**File:** `pkg/postgres/datasource.postgres.go:779`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sanitized:** No

#### flow\_3f5a6cab01d341d7: http\_body -> template
**File:** `pkg/postgres/datasource.postgres.go:779`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_655c35eccc10ec1c: http\_body -> template
**File:** `pkg/postgres/datasource.postgres.go:785`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_b703f68f8fc2c70e: http\_body -> template
**File:** `pkg/sqlserver/datasource.sqlserver.go:572`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sanitized:** No

#### flow\_d6b67ccb78d79e8b: http\_body -> template
**File:** `pkg/sqlserver/datasource.sqlserver.go:572`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonMap\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No

#### flow\_01a9816181d2c1bd: http\_body -> template
**File:** `pkg/sqlserver/datasource.sqlserver.go:578`
**Risk:** high
**Notes:** Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Source:** `if err := json.Unmarshal\(byteData, &jsonArray\); err == nil {`
**Sink:** `if err := json.Unmarshal\(byteData, &jsonString\); err == nil {`
**Sanitized:** No


### Medium Risk Flows (0)



## Focus Areas

Based on analysis, pay special attention to:

1. **Unsanitized High-Risk Flows** - 41 data flows without sanitization
2. **Critical Security Findings** - 1 high/critical security issues detected

