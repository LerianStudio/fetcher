# Security Data Flow Analysis

**Generated:** 2026-03-12T10:53:52-03:00

## Executive Summary

| Metric | Value |
|--------|-------|
| Languages Analyzed | 1 |
| Total Sources | 114 |
| Total Sinks | 49 |
| Total Flows | 99 |
| Unsanitized Flows | 99 |
| Critical Risk Flows | 0 |
| High Risk Flows | 41 |
| Nil/Null Risks | 114 |

## Risk Assessment

### HIGH (41 issues)

High-risk security issues that should be addressed promptly.

### NIL SAFETY (114 issues)

Unchecked nil/null values that may cause runtime panics or crashes.

## Analysis Limits

- Data flow tracking is heuristic and primarily intra-file; flows that cross function boundaries may be under-reported.

## Critical & High Risk Flows

### HIGH Flow 1: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:252` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:265` (Function: h.GetQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
conn, err := h.GetQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 2: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:252` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:330` (Function: h.TestQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.TestQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 3: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:252` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:419` (Function: h.UpdateCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
conn, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 4: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:252` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:480` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 5: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:252` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:615` (Function: h.GetSchemaQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.GetSchemaQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 6: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:315` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:330` (Function: h.TestQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.TestQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 7: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:315` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:419` (Function: h.UpdateCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
conn, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 8: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:315` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:480` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 9: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:315` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:615` (Function: h.GetSchemaQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.GetSchemaQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 10: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:380` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:419` (Function: h.UpdateCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
conn, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 11: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:380` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:480` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 12: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:380` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:615` (Function: h.GetSchemaQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.GetSchemaQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 13: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:467` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:480` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 14: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:467` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:615` (Function: h.GetSchemaQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.GetSchemaQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 15: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/connection.go:600` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/connection.go:615` (Function: h.GetSchemaQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
resp, err := h.GetSchemaQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 16: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/fetcher.go:190` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/fetcher.go:205` (Function: h.GetJobQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
job, err := h.GetJobQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 17: Data from http\_path \(Fiber path parameter\) in variable 'connectionID' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/migration.go:140` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/migration.go:187` (Function: h.AssignCmd.Execute)

**Sanitized:** No

**Source Context:**
```
connectionID, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
conn, err := h.AssignCmd.Execute(ctx, orgID, connectionID, productID)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 18: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/product.go:228` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/product.go:241` (Function: h.GetQuery.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
product, err := h.GetQuery.Execute(ctx, orgID, id)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 19: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/product.go:228` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/product.go:331` (Function: h.UpdateCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
product, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 20: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/product.go:228` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/product.go:392` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 21: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/product.go:292` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/product.go:331` (Function: h.UpdateCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
product, err := h.UpdateCmd.Execute(ctx, orgID, id, request)
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 22: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/product.go:292` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/product.go:392` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 23: Data from http\_path \(Fiber path parameter\) in variable 'id' flows to template \(Template execution\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `components/manager/internal/adapters/http/in/product.go:379` (Type: http\_path)

**Sink:** `components/manager/internal/adapters/http/in/product.go:392` (Function: h.DeleteCmd.Execute)

**Sanitized:** No

**Source Context:**
```
id, err := uuid.Parse(c.Params("id"))
```

**Sink Context:**
```
if err := h.DeleteCmd.Execute(ctx, orgID, id); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 24: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/itestkit/addons/queuekit/assertions.go:251` (Type: http\_body)

**Sink:** `pkg/itestkit/addons/queuekit/assertions.go:256` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(expected, &e); err != nil {
```

**Sink Context:**
```
if err := json.Unmarshal(actual, &a); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 25: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/itestkit/addons/queuekit/assertions.go:251` (Type: http\_body)

**Sink:** `pkg/itestkit/addons/queuekit/assertions.go:278` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(expected, &e); err != nil {
```

**Sink Context:**
```
if err := json.Unmarshal(data, &m); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 26: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/itestkit/addons/queuekit/assertions.go:256` (Type: http\_body)

**Sink:** `pkg/itestkit/addons/queuekit/assertions.go:278` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(actual, &a); err != nil {
```

**Sink Context:**
```
if err := json.Unmarshal(data, &m); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 27: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/itestkit/addons/queuekit/matcher.go:153` (Type: http\_body)

**Sink:** `pkg/itestkit/addons/queuekit/matcher.go:167` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(msg.Body, &data); err != nil {
```

**Sink Context:**
```
if err := json.Unmarshal(msg.Body, &data); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 28: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/itestkit/addons/queuekit/matcher.go:153` (Type: http\_body)

**Sink:** `pkg/itestkit/addons/queuekit/matcher.go:181` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(msg.Body, &data); err != nil {
```

**Sink Context:**
```
if err := json.Unmarshal(msg.Body, &data); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 29: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/itestkit/addons/queuekit/matcher.go:167` (Type: http\_body)

**Sink:** `pkg/itestkit/addons/queuekit/matcher.go:181` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(msg.Body, &data); err != nil {
```

**Sink Context:**
```
if err := json.Unmarshal(msg.Body, &data); err != nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 30: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/mysql/datasource.mysql.go:375` (Type: http\_body)

**Sink:** `pkg/mysql/datasource.mysql.go:381` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 31: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/mysql/datasource.mysql.go:375` (Type: http\_body)

**Sink:** `pkg/mysql/datasource.mysql.go:387` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 32: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/mysql/datasource.mysql.go:381` (Type: http\_body)

**Sink:** `pkg/mysql/datasource.mysql.go:387` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 33: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/oracle/datasource.oracle.go:629` (Type: http\_body)

**Sink:** `pkg/oracle/datasource.oracle.go:635` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 34: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/oracle/datasource.oracle.go:629` (Type: http\_body)

**Sink:** `pkg/oracle/datasource.oracle.go:641` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 35: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/sqlserver/datasource.sqlserver.go:578` (Type: http\_body)

**Sink:** `pkg/sqlserver/datasource.sqlserver.go:584` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 36: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/sqlserver/datasource.sqlserver.go:572` (Type: http\_body)

**Sink:** `pkg/sqlserver/datasource.sqlserver.go:584` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 37: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/sqlserver/datasource.sqlserver.go:572` (Type: http\_body)

**Sink:** `pkg/sqlserver/datasource.sqlserver.go:578` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 38: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/postgres/datasource.postgres.go:785` (Type: http\_body)

**Sink:** `pkg/postgres/datasource.postgres.go:791` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 39: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/postgres/datasource.postgres.go:779` (Type: http\_body)

**Sink:** `pkg/postgres/datasource.postgres.go:791` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 40: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/postgres/datasource.postgres.go:779` (Type: http\_body)

**Sink:** `pkg/postgres/datasource.postgres.go:785` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonMap); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

### HIGH Flow 41: Data from http\_body \(JSON decode from request\) in variable 'err' flows to template \(JSON deserialization\) \[unsanitized - potential vulnerability\]

**Risk Level:** High

**Source:** `pkg/oracle/datasource.oracle.go:635` (Type: http\_body)

**Sink:** `pkg/oracle/datasource.oracle.go:641` (Function: json.Unmarshal)

**Sanitized:** No

**Source Context:**
```
if err := json.Unmarshal(byteData, &jsonArray); err == nil {
```

**Sink Context:**
```
if err := json.Unmarshal(byteData, &jsonString); err == nil {
```

**Recommendation:** Ensure template engine auto-escapes

---

## Nil/Null Safety Issues

| File | Line | Variable | Origin | Checked | Risk |
|------|------|----------|--------|---------|------|
| components/manager/internal/services/command/create\_fetcher\_job.go | 389 | message | map\_lookup | No | High |
| components/manager/internal/services/command/create\_fetcher\_job.go | 407 | header | map\_lookup | No | Medium |
| components/manager/internal/services/command/create\_fetcher\_job.go | 231 | found | map\_lookup | No | Medium |
| components/manager/internal/services/query/get\_connection\_schema.go | 125 | fields | lookup\_operation | No | Medium |
| components/manager/internal/services/query/get\_connection\_schema.go | 263 | systemSchemas | map\_lookup | No | High |
| components/manager/internal/services/query/get\_connection\_schema.go | 226 | systemTables | map\_lookup | No | High |
| components/manager/internal/services/query/get\_connection\_schema.go | 290 | systemDatabases | map\_lookup | No | High |
| components/manager/internal/services/query/validate\_schema.go | 138 | found | map\_lookup | No | Medium |
| components/manager/internal/services/query/validate\_schema.go | 149 | tables | lookup\_operation | No | Medium |
| components/manager/internal/services/query/validate\_schema.go | 276 | Tables | map\_lookup | No | Medium |
| components/manager/internal/services/query/validate\_schema.go | 441 | exists | map\_lookup | No | Medium |
| components/manager/internal/services/query/validate\_schema.go | 100 | configNames | lookup\_operation | No | Medium |
| components/manager/internal/services/query/validate\_schema.go | 164 | schemas | lookup\_operation | No | High |
| components/manager/internal/services/query/validate\_schema.go | 35 | pluginCRMTableMapping | map\_lookup | No | High |
| components/worker/internal/services/extract-data.go | 528 | encryptSecretKey | lookup\_operation | No | Medium |
| components/worker/internal/services/extract-data.go | 533 | hashSecretKey | lookup\_operation | No | Medium |
| components/worker/internal/services/extract-data.go | 301 | errorMetadata | map\_lookup | No | Medium |
| components/worker/internal/services/extract-data.go | 190 | exists | map\_lookup | No | Medium |
| components/worker/internal/services/extract-data.go | 256 | orgMatches | lookup\_operation | No | High |
| components/worker/internal/services/extract-data.go | 198 | existOrg | map\_lookup | No | Medium |
| components/worker/internal/services/extract-data.go | 249 | matches | lookup\_operation | No | High |
| components/worker/internal/services/extract-data.go | 407 | databaseExists | map\_lookup | No | Medium |
| components/worker/internal/services/extract\_crm\_data.go | 314 | encryptedFields | map\_lookup | No | High |
| components/worker/internal/services/extract\_crm\_data.go | 452 | exists | map\_lookup | No | Medium |
| components/worker/internal/services/extract\_crm\_data.go | 279 | encryptSecretKey | lookup\_operation | No | Medium |
| components/worker/internal/services/extract\_crm\_data.go | 277 | hashSecretKey | lookup\_operation | No | Medium |
| components/worker/internal/services/extract\_crm\_data.go | 180 | fieldMappings | map\_lookup | No | High |
| pkg/datasource/datasource-factory.go | 33 | defaultDataSourceConfigBuilders | map\_lookup | No | Medium |
| pkg/datasource/sslmode/mongodb.go | 23 | validMongoDBModes | map\_lookup | No | High |
| pkg/datasource/sslmode/mongodb.go | 45 | valid | map\_lookup | No | Medium |
| pkg/datasource/sslmode/mysql.go | 22 | validMySQLModes | map\_lookup | No | High |
| pkg/datasource/sslmode/mysql.go | 43 | valid | map\_lookup | No | Medium |
| pkg/datasource/sslmode/oracle.go | 25 | validOracleModes | map\_lookup | No | High |
| pkg/datasource/sslmode/oracle.go | 48 | valid | map\_lookup | No | Medium |
| pkg/datasource/sslmode/postgresql.go | 21 | validPostgreSQLModes | map\_lookup | No | High |
| pkg/datasource/sslmode/postgresql.go | 44 | valid | map\_lookup | No | Medium |
| pkg/datasource/sslmode/sqlserver.go | 22 | validSQLServerModes | map\_lookup | No | High |
| pkg/datasource/sslmode/sqlserver.go | 43 | valid | map\_lookup | No | Medium |
| pkg/itestkit/addons/e2ekit/build\_secrets.go | 134 | value | lookup\_operation | No | Medium |
| pkg/itestkit/addons/metricskit/metrics.go | 306 | result | map\_lookup | No | Medium |
| pkg/itestkit/addons/metricskit/reporter.go | 62 | errorCounts | lookup\_operation | No | Medium |
| pkg/itestkit/addons/queuekit/assertions.go | 251 | e | json\_unmarshal | No | High |
| pkg/itestkit/addons/queuekit/assertions.go | 256 | a | json\_unmarshal | No | High |
| pkg/itestkit/addons/queuekit/assertions.go | 278 | m | json\_unmarshal | No | High |
| pkg/itestkit/addons/queuekit/consumer.go | 249 | parsed | map\_lookup | No | Medium |
| pkg/itestkit/addons/queuekit/consumer.go | 392 | consumer | map\_lookup | No | High |
| pkg/itestkit/addons/queuekit/consumer.go | 210 | result | map\_lookup | No | High |
| pkg/itestkit/addons/queuekit/matcher.go | 190 | v | type\_assertion | No | Medium |
| pkg/itestkit/addons/queuekit/matcher.go | 213 | current | map\_lookup | No | High |
| pkg/itestkit/addons/queuekit/matcher.go | 229 | exists | map\_lookup | No | Medium |
| pkg/itestkit/addons/queuekit/matcher.go | 258 | e | type\_assertion | No | Medium |
| pkg/itestkit/addons/queuekit/matcher.go | 181 | data | json\_unmarshal | No | Medium |
| pkg/itestkit/chaos\_toxiproxy.go | 58 | NetworkAliases | map\_lookup | No | Medium |
| pkg/itestkit/container\_generic.go | 79 | Containers | map\_lookup | No | High |
| pkg/itestkit/customizer\_options.go | 112 | NetworkAliases | map\_lookup | No | High |
| pkg/itestkit/hostport.go | 31 | override | lookup\_operation | No | Medium |
| pkg/itestkit/infra.go | 20 | seen | map\_lookup | No | High |
| pkg/itestkit/infra.go | 36 | exists | map\_lookup | No | Medium |
| pkg/itestkit/infra/oracle/oracle.go | 92 | NetworkAliases | map\_lookup | No | Medium |
| pkg/model/datasource/oracle/datasource-config.go | 65 | schemas | lookup\_operation | No | High |
| pkg/model/datasource/postgres/datasource-config.go | 65 | schemas | lookup\_operation | No | High |
| pkg/model/datasource/postgres/datasource-config.go | 169 | exists | map\_lookup | No | Medium |
| pkg/model/datasource/sqlserver/datasource-config.go | 68 | schemas | lookup\_operation | No | High |
| pkg/model/datasource/sqlserver/datasource-config.go | 126 | exists | map\_lookup | No | Medium |
| pkg/mongodb/connection/connection.mongodb.go | 83 | config | map\_lookup | No | High |
| pkg/mongodb/connection/connection.mongodb.go | 745 | opts | lookup\_operation | No | Medium |
| pkg/mongodb/connection/connection.mongodb.go | 446 | errFind | lookup\_operation | No | Medium |
| pkg/mongodb/datasource.mongodb.go | 206 | v | type\_assertion | No | High |
| pkg/mongodb/datasource.mongodb.go | 323 | dataType | map\_lookup | No | Medium |
| pkg/mongodb/datasource.mongodb.go | 516 | exists | map\_lookup | No | Medium |
| pkg/mongodb/datasource.mongodb.go | 559 | typeHierarchy | map\_lookup | No | High |
| pkg/mongodb/datasource.mongodb.go | 839 | regexPattern | map\_lookup | No | Medium |
| pkg/mongodb/datasource.mongodb.go | 655 | findOptions | lookup\_operation | No | High |
| pkg/mongodb/datasource.mongodb.go | 575 | newLevel | map\_lookup | No | Medium |
| pkg/mongodb/datasource.mongodb.go | 576 | currentLevel | map\_lookup | No | Medium |
| pkg/mongodb/datasource.mongodb.go | 869 | singleValueOps | map\_lookup | No | Medium |
| pkg/mongodb/job/job.mongodb.go | 80 | config | map\_lookup | No | High |
| pkg/mongodb/job/job.mongodb.go | 430 | opts | lookup\_operation | No | High |
| pkg/mongodb/mongo.go | 33 | exists | map\_lookup | No | Medium |
| pkg/mongodb/product/product.mongodb.go | 216 | opts | lookup\_operation | No | High |
| pkg/mongodb/product/product.mongodb.go | 82 | config | map\_lookup | No | High |
| pkg/mysql/datasource.mysql.go | 314 | exists | map\_lookup | No | Medium |
| pkg/mysql/datasource.mysql.go | 381 | jsonArray | json\_unmarshal | No | Medium |
| pkg/mysql/datasource.mysql.go | 761 | singleValueOps | map\_lookup | No | Medium |
| pkg/mysql/datasource.mysql.go | 315 | IsPrimaryKey | map\_lookup | No | Medium |
| pkg/mysql/datasource.mysql.go | 375 | jsonMap | json\_unmarshal | No | Medium |
| pkg/mysql/datasource.mysql.go | 387 | jsonString | json\_unmarshal | No | Medium |
| pkg/mysql/mysql.go | 82 | exists | map\_lookup | No | Medium |
| pkg/oracle/datasource.oracle.go | 635 | jsonArray | json\_unmarshal | No | Medium |
| pkg/oracle/datasource.oracle.go | 641 | jsonString | json\_unmarshal | No | Medium |
| pkg/oracle/datasource.oracle.go | 1054 | singleValueOps | map\_lookup | No | Medium |
| pkg/oracle/datasource.oracle.go | 569 | exists | map\_lookup | No | Medium |
| pkg/oracle/datasource.oracle.go | 570 | IsPrimaryKey | map\_lookup | No | Medium |
| pkg/oracle/datasource.oracle.go | 629 | jsonMap | json\_unmarshal | No | Medium |
| pkg/oracle/oracle.go | 90 | exists | map\_lookup | No | Medium |
| pkg/postgres/datasource.postgres.go | 785 | jsonArray | json\_unmarshal | No | Medium |
| pkg/postgres/datasource.postgres.go | 791 | jsonString | json\_unmarshal | No | Medium |
| pkg/postgres/datasource.postgres.go | 1410 | singleValueOps | map\_lookup | No | Medium |
| pkg/postgres/datasource.postgres.go | 655 | exists | map\_lookup | No | Medium |
| pkg/postgres/datasource.postgres.go | 656 | IsPrimaryKey | map\_lookup | No | Medium |
| pkg/postgres/datasource.postgres.go | 779 | jsonMap | json\_unmarshal | No | Medium |
| pkg/rabbitmq/rabbitmq.go | 1158 | v | type\_assertion | No | High |
| pkg/rabbitmq/rabbitmq.go | 967 | found | map\_lookup | No | Medium |
| pkg/redis/factory.go | 44 | redisCache | map\_lookup | No | Medium |
| pkg/redis/redis\_cache.go | 86 | value | json\_unmarshal | No | Medium |
| pkg/sqlserver/datasource.sqlserver.go | 511 | exists | map\_lookup | No | Medium |
| pkg/sqlserver/datasource.sqlserver.go | 512 | IsPrimaryKey | map\_lookup | No | Medium |
| pkg/sqlserver/datasource.sqlserver.go | 572 | jsonMap | json\_unmarshal | No | Medium |
| pkg/sqlserver/datasource.sqlserver.go | 578 | jsonArray | json\_unmarshal | No | Medium |
| pkg/sqlserver/datasource.sqlserver.go | 584 | jsonString | json\_unmarshal | No | Medium |
| pkg/sqlserver/datasource.sqlserver.go | 977 | singleValueOps | map\_lookup | No | Medium |
| pkg/sqlserver/sqlserver.go | 82 | exists | map\_lookup | No | Medium |
| tests/shared/assertions.go | 133 | err | lookup\_operation | No | Medium |
| tests/shared/infra.go | 34 | val | lookup\_operation | No | Medium |

## Language Breakdown

### Go

| Metric | Value |
|--------|-------|
| Sources | 114 |
| Sinks | 49 |
| Flows | 99 |
| Unsanitized | 99 |
| Critical | 0 |
| High | 41 |
| Nil Risks | 114 |

## General Recommendations

1. **Input Validation**: Always validate and sanitize user input at the entry point.
2. **Parameterized Queries**: Use prepared statements or parameterized queries for all database operations.
3. **Output Encoding**: Encode output appropriately for the context (HTML, URL, JavaScript).
4. **Nil Checks**: Always check for nil/null before dereferencing pointers or optional values.
5. **Principle of Least Privilege**: Avoid command execution; if required, use strict allow lists.
6. **Security Testing**: Integrate security scanning into CI/CD pipelines for continuous monitoring.
