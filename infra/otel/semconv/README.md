# semconv

Webitel's semantic-convention registry, and the Go package generated from it.

One place fixes metric names, units, required attributes and histogram
boundaries. Services import the generated package instead of inventing names
locally.

## Why this is not a hand-written Go package any more

The package this replaced was 96 lines of hand-written `attribute.Key`
constants pinned to one upstream version. Nothing checked it, it drifted from
upstream invisibly, and it contained only what somebody had got around to
adding. Eight of its keys sat in namespaces that were not ours — `db.sql_state`,
`db.rows_affected` and `db.sql_table` in upstream's `db.*`, a bare `name` key,
and a top-level `message.*` block.

## Layout

| Path | What it is |
|---|---|
| `registry/manifest.yaml` | Registry metadata, **and the upstream version pin** |
| `registry/db.yaml` | Database conventions, and the WTEL-10157 migration table |
| `registry/rpc.yaml` | RPC conventions |
| `registry/resource.yaml` | Resource attributes every service must set |
| `templates/go/weaver.yaml` | Generator config: namespace, histogram boundaries, acronyms |
| `templates/go/attribute.go.j2` | The Go template |
| `attribute.go` | **Generated. Do not edit.** |
| `generate.sh` | Regenerates `attribute.go` |

## Regenerating

```sh
./generate.sh            # regenerate attribute.go
./generate.sh --check    # fail if attribute.go is out of date (this is what CI runs)
```

`generate.sh` downloads a pinned [Weaver](https://github.com/open-telemetry/weaver)
release binary into `~/.cache/webitel-weaver/` and verifies its published
SHA-256 first. Weaver is written in Rust, but no Rust toolchain is needed — the
release ships prebuilt binaries, and `generate.sh` picks the one for the host.
`attribute.go` is committed, so nothing downstream needs Weaver to build.

## Rules the registry enforces

- **Names are OTel-native**, not Prometheus-style. Ten services already run
  `otelgrpc`, which emits upstream names, and a second naming style beside it
  would put two dictionaries in one pipeline. This holds even once we emit the
  names ourselves — see below.
- **Our own durations are seconds.** Upstream moved at semconv v1.21.0. Two
  places in this fleet still emit milliseconds and are *not* the example to
  copy: the vendored `otelhttp` copy in `webitel.go`, frozen at semconv v1.20.0,
  and `infra/sqldb/sql/sql.go`, which declares `unitMilliseconds` beside
  `db.sql.client.latency` — an upstream namespace holding a non-upstream name,
  with the unit baked into the identifier.

  One caveat, because it is a trap: imported upstream metrics keep whatever unit
  upstream declares. `db.*` are seconds, but `rpc.client.duration` and
  `rpc.server.duration` are still **milliseconds** at v1.30.0. We take
  cross-cutting conventions unchanged, so those two stay `ms` — never give them
  the second-based boundaries in `templates/go/weaver.yaml`.
- **New conventions ship at `development` stability.**
- **No tenant or user identifiers in metric attributes.** `domain_id` is fine in
  a trace; in a metric it is uncontrolled cardinality.
- **Every attribute carries a requirement level** — `required`, `recommended`,
  `opt_in`, or `conditionally_required`. Without one, a convention set
  guarantees nothing.
- **Nothing of ours goes in an upstream namespace.** This is the rule the old
  package broke.

### Our own instrumentation is the direction of travel

The RPC conventions here are declared as a **contract**, not as a description of
what `otelgrpc` happens to emit. Webitel intends to write its own instrumentation
rather than depend on a third party for naming decisions (WTEL-10157). That work
is explicitly future — this registry is what it will implement, which is why the
RPC group is spelled out in full even though nothing of ours emits it yet.

Cross-cutting conventions — RPC, SQL, HTTP, runtime — are taken from upstream
unchanged and referenced, never redefined. `db.client.connection.*` already
covers connection pools. The `imports.metrics` block in `registry/registry.yaml`
pulls in the 20 upstream metrics Webitel services emit, so weaver validates the
names exist at the pinned version — a typo or an upstream rename fails the
build instead of producing a metric nobody graphs. Domain telemetry that upstream does not cover (dropped
WebSocket events, live FreeSWITCH nodes, queue depth, call-attempt states)
cannot be enumerated in advance, so the registry works as a gate: a domain
metric enters when it is written, but it does not appear without passing
through here.

## Adding an enum value

Upstream defines 41 database systems and 5 RPC systems. We generate only the
ones this fleet speaks — `postgresql` and `grpc` — because the rest are dead
constants nothing can reach. When a service starts talking to something else,
add the value to `enum_values` in `templates/go/weaver.yaml` and regenerate:

```yaml
enum_values:
  db.system.name: [postgresql, redis]
```

## The namespace: `com.webitel.*`

Conventions that stay ours use **reverse-domain** `com.webitel.*`, decided on
WTEL-10157 — follow the OTel spec directly so we stay consistent with it, and
say so explicitly here.

The [naming spec](https://opentelemetry.io/docs/specs/semconv/general/naming/)
recommends reverse-domain for names that may be seen outside the company, and
permits a plain application-name prefix only for purely internal systems.
Webitel runs on customers' machines beside other vendors' software, so the
reverse-domain form is the one the spec is actually pointing at.

The cost is real and worth stating: Prometheus turns dots into underscores, so
`com.webitel.db.user` is scraped as `com_webitel_db_user`. That is longer than
`webitel_db_user` and it lands on
[WTEL-10159](https://webitel.atlassian.net/browse/WTEL-10159), which adds the
Prometheus exporter. It is accepted, not overlooked.

The prefix is written in exactly one place, `templates/go/weaver.yaml`, and the
generated package exposes it as `semconv.Namespace`. To change it everywhere:

```sh
./generate.sh --set-namespace com.webitel
```

Go identifiers do not move when the prefix does — `WebitelDBUserKey` keeps its
name and only its value changes — so consumers keep compiling. The command
refuses a namespace that collides with one upstream owns (`db`, `rpc`,
`service`, …): that is the mistake this registry exists to stop, and it would
make the rewrite ambiguous.

## Raising the upstream pin

Change the single `dependencies[0]` entry in `registry/manifest.yaml` and
regenerate. The current pin is **v1.30.0**, the minimum WTEL-10157 requires:
`db.response.status_code` landed in v1.28.0 and `db.response.returned_rows` in
v1.30.0.

Note that `generate.sh` deliberately does **not** pass Weaver's `--future`
flag. It applies the newest validation rules to the dependency as well, and
upstream v1.30.0's own model does not satisfy them — it turns a valid registry
into 40+ hard errors. Re-evaluate when the pin is raised.

## The WTEL-10157 migration

Only the first row is an upstream rename. The rest were ours, created when no
upstream analog existed; moving them was our decision, not a deprecation.

| Was | Is now | |
|---|---|---|
| `db.statement` | `db.query.text` | genuine upstream rename |
| `db.sql_state` | `db.response.status_code` | needs semconv ≥ v1.28.0 |
| `db.rows_affected` | `db.response.returned_rows` **on SELECT only** | needs semconv ≥ v1.30.0 |
| `db.rows_affected` | `com.webitel.db.rows_affected` on writes and COPY | see below |
| `db.sql_table` | `db.collection.name` | permitted — see below |
| `db.batch.size` | `db.operation.batch.size` | upstream analog exists |
| `db.query.parameters` | `db.operation.parameter.<key>` | template attribute |
| `db.prepare_stmt.name` | `com.webitel.db.prepare_stmt.name` | stays ours |
| `db.user` | `com.webitel.db.user` | stays ours; upstream removed its own |
| `message.*` | `rpc.message.*` | upstream already defines all four |
| `name` | *dropped* | only ever held the constant `"message"` |

The `db.rows_affected` row is a deliberate deviation from the ticket, which
asked for a straight move to `db.response.returned_rows`. Upstream defines that
as *rows the operation returned*, and pgx reports one number for everything —
rows returned for `SELECT`, rows changed for a write, rows copied for `COPY`.
Labelling an `UPDATE` that touched 500 rows as having *returned* 500 rows is
simply false, and upstream has no attribute for rows affected. So `SELECT` maps
to the upstream key and everything else keeps a Webitel one.

`db.collection.name` is only permitted when the value is not extracted from
query text. Webitel sets it on `CopyFrom` spans only, where the value comes from
`pgx.TraceCopyFromStartData.TableName` — a structured `pgx.Identifier` handed in
by the caller — so the condition is satisfied.
