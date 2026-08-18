package semconv

import (
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

// Derived from the generated package, so a namespace flip does not need this
// file edited — only the suffixes below are pinned.
var ourNamespace = Namespace + "."

// Namespaces owned by OpenTelemetry. Ours must never appear in them.
var upstreamNamespaces = []string{
	"db.", "rpc.", "http.", "net.", "network.", "client.", "server.",
	"service.", "messaging.", "url.", "user_agent.", "error.",
}

// Pins the wire names. These reach dashboards and alert rules, so changing one
// is a breaking change and must be made here as well as in the registry.
func TestKeyValues(t *testing.T) {
	cases := []struct {
		key  attribute.Key
		want string
	}{
		{WebitelDBPrepareStmtNameKey, ourNamespace + "db.prepare_stmt.name"},
		{WebitelDBUserKey, ourNamespace + "db.user"},

		{DBQueryTextKey, "db.query.text"},                        // was db.statement
		{DBResponseStatusCodeKey, "db.response.status_code"},     // was db.sql_state
		{DBResponseReturnedRowsKey, "db.response.returned_rows"}, // was db.rows_affected
		{DBCollectionNameKey, "db.collection.name"},              // was db.sql_table
		{DBOperationBatchSizeKey, "db.operation.batch.size"},     // was db.batch.size
		{DBOperationParameterKey, "db.operation.parameter"},      // was db.query.parameters
		{DBSystemNameKey, "db.system.name"},

		{RPCMessageTypeKey, "rpc.message.type"},
		{RPCMessageIDKey, "rpc.message.id"},
		{RPCMessageCompressedSizeKey, "rpc.message.compressed_size"},
		{RPCMessageUncompressedSizeKey, "rpc.message.uncompressed_size"},
		{RPCGRPCStatusCodeKey, "rpc.grpc.status_code"},

		{ServiceNameKey, "service.name"},
		{ServiceInstanceIDKey, "service.instance.id"},
		{ServiceNamespaceKey, "service.namespace"},
	}

	for _, c := range cases {
		if got := string(c.key); got != c.want {
			t.Errorf("key = %q, want %q", got, c.want)
		}
	}
}

// Namespace itself must not sit in a namespace upstream owns — generate.sh
// refuses such a value, and this catches a file edited around it.
func TestNamespaceDoesNotCollideWithUpstream(t *testing.T) {
	for _, ns := range upstreamNamespaces {
		if strings.HasPrefix(Namespace+".", ns) {
			t.Errorf("Namespace %q collides with the upstream namespace %q", Namespace, ns)
		}
	}
}

// The rule the hand-written package broke.
func TestOurKeysAreNotInUpstreamNamespaces(t *testing.T) {
	ours := []attribute.Key{
		WebitelDBPrepareStmtNameKey,
		WebitelDBUserKey,
	}

	for _, k := range ours {
		name := string(k)
		if !strings.HasPrefix(name, ourNamespace) {
			t.Errorf("%q is one of ours but does not carry the %q prefix", name, ourNamespace)
		}
		for _, ns := range upstreamNamespaces {
			if strings.HasPrefix(name, ns) {
				t.Errorf("%q is one of ours but sits in the upstream namespace %q", name, ns)
			}
		}
	}
}

func TestWellKnownValues(t *testing.T) {
	if got := DBSystemNamePostgresql; got.Key != DBSystemNameKey || got.Value.AsString() != "postgresql" {
		t.Errorf("DBSystemNamePostgresql = %v, want %s=postgresql", got, DBSystemNameKey)
	}

	if got := RPCSystemGRPC; got.Key != RPCSystemKey || got.Value.AsString() != "grpc" {
		t.Errorf("RPCSystemGRPC = %v, want %s=grpc", got, RPCSystemKey)
	}
}

// Guards the pin: below v1.30.0 two attributes pgx sets stop existing.
func TestSchemaURLs(t *testing.T) {
	if SchemaURL == "" {
		t.Error("SchemaURL is empty")
	}

	const wantUpstream = "https://opentelemetry.io/schemas/1.30.0"
	if UpstreamSchemaURL != wantUpstream {
		t.Errorf("UpstreamSchemaURL = %q, want %q — if the pin was raised deliberately, update this test and the registry together",
			UpstreamSchemaURL, wantUpstream)
	}
}
