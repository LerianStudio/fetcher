package query

import (
	"strings"
	"testing"

	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/v2/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/v2/pkg/model"
)

// TestOracleCrossPathCasingParity is the load-bearing cross-path guard for the
// UPPERCASE-CANONICAL Oracle contract. It asserts that the identity the MANAGER
// validation path resolves (normalizeTableNameForValidation /
// normalizeFieldNameForValidation) is byte-identical to the identity the WORKER
// extraction path produces (tablenorm.NormalizeTable / NormalizeField), for the same
// caller-supplied table/field strings — and that both equal the physical UPPERCASE
// data-key identity.
//
// Both paths fold Oracle to UPPERCASE. If a future change makes one path diverge from
// the other (e.g. one reverts to lowercase), this test FAILS — which is the whole
// point: the Manager must validate the SAME identity the Worker extracts AND the
// SAME identity the extracted rows are keyed by, or a request that validates 200 OK
// will silently extract nothing (or extract data a consumer cannot read by key).
func TestOracleCrossPathCasingParity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		table string
		field string
		// physicalTable/physicalField are the UPPERCASE keys the extracted result rows
		// actually carry (pkg/oracle.createRowMap keys verbatim by the physical catalog
		// columns). The canonical normalization MUST equal these so snapshot == data.
		physicalTable string
		physicalField string
	}{
		{name: "lowercase request", table: "accounts", field: "balance", physicalTable: "ACCOUNTS", physicalField: "BALANCE"},
		{name: "uppercase request", table: "ACCOUNTS", field: "BALANCE", physicalTable: "ACCOUNTS", physicalField: "BALANCE"},
		{name: "mixed request", table: "Accounts", field: "AccountId", physicalTable: "ACCOUNTS", physicalField: "ACCOUNTID"},
		{name: "owner-qualified request", table: "hr.employees", field: "salary", physicalTable: "HR.EMPLOYEES", physicalField: "SALARY"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			managerTable := normalizeTableNameForValidation(tc.table, model.TypeOracle)
			workerTable := tablenorm.NormalizeTable(model.TypeOracle, tc.table)
			if managerTable != workerTable {
				t.Fatalf("Oracle table identity diverged: Manager %q != Worker %q (input %q)",
					managerTable, workerTable, tc.table)
			}

			managerField := normalizeFieldNameForValidation(tc.field, model.TypeOracle)
			workerField := tablenorm.NormalizeField(model.TypeOracle, tc.field)
			if managerField != workerField {
				t.Fatalf("Oracle field identity diverged: Manager %q != Worker %q (input %q)",
					managerField, workerField, tc.field)
			}

			// The canonical identity MUST equal the physical UPPERCASE data key, so the
			// snapshot/validation identity matches what the extracted rows are keyed by.
			if managerTable != tc.physicalTable {
				t.Fatalf("Manager Oracle table identity %q != physical data key %q", managerTable, tc.physicalTable)
			}
			if managerField != tc.physicalField {
				t.Fatalf("Manager Oracle field identity %q != physical data key %q", managerField, tc.physicalField)
			}
		})
	}
}

// TestOracleParityResolvesAgainstUppercaseSnapshot proves the parity is not academic:
// the canonicalized request identity actually resolves against the snapshot the
// Manager/Worker build from a LOWERCASE GetSchemaInfo-style schema (HasTable/HasField
// succeed) regardless of the request's original case, AND the resolved key equals the
// physical UPPERCASE data key. This binds the GetSchemaInfo output, the snapshot
// builder normalization, the request normalization, and the data-key casing into one
// assertion.
func TestOracleParityResolvesAgainstUppercaseSnapshot(t *testing.T) {
	t.Parallel()

	// GetSchemaInfo lowercases what it returns, so the discovered schema is lowercase;
	// the snapshot builder re-folds it to UPPERCASE for Oracle.
	discovered := model.NewDataSourceSchema("ora-main")
	discovered.AddTable("accounts", []string{"id", "balance"})

	snapshot := schemacompat.BuildSnapshot("ora-main", model.TypeOracle, discovered, schemacompat.SnapshotOptions{
		FilterSystemTables: true,
		Normalize:          true,
	})

	// The snapshot MUST be UPPERCASE (== physical data keys).
	roundTrip := schemacompat.DataSourceSchemaFromSnapshot(snapshot)
	if !roundTrip.HasTable("ACCOUNTS") {
		t.Fatalf("snapshot table is not UPPERCASE; tables=%v", roundTrip.Tables)
	}
	if roundTrip.HasTable("accounts") {
		t.Fatalf("snapshot must NOT carry the lowercase table key")
	}

	// A mixed-case request must resolve once normalized by EITHER path (they are equal,
	// per the parity test above), and resolve to the UPPERCASE physical identity.
	reqTable := tablenorm.NormalizeTable(model.TypeOracle, "Accounts")
	reqField := tablenorm.NormalizeField(model.TypeOracle, "Balance")

	if reqTable != "ACCOUNTS" || reqField != "BALANCE" {
		t.Fatalf("normalized request not UPPERCASE: table=%q field=%q", reqTable, reqField)
	}
	if !roundTrip.HasTable(reqTable) {
		t.Fatalf("normalized Oracle table %q did not resolve against UPPERCASE snapshot %v", reqTable, roundTrip.Tables)
	}
	if !roundTrip.HasField(reqTable, reqField) {
		t.Fatalf("normalized Oracle field %q did not resolve in table %q", reqField, reqTable)
	}
	if !strings.EqualFold(reqTable, "accounts") { // sanity: same logical table, UPPER form
		t.Fatalf("unexpected table identity %q", reqTable)
	}
}
