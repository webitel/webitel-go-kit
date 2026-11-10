# appconfig/secret

Keeps passwords, tokens, DSNs and private keys out of plain-text config files
and process environments. A secret config value can be read from:

- a **systemd credential** — encrypted at rest with `systemd-creds`, decrypted
  by systemd at service start into a private, non-swappable in-memory
  directory that only the service can read;
- a **file** — e.g. a Docker or Kubernetes secret mounted under `/run/secrets`.

Plain literal values keep working, so services and deployments can migrate one
key at a time.

Services built on `appconfig.Loader` get this for `postgres.dsn`,
`redis.password` and `pubsub.url` with no code change; other keys are one
`MarkSecret` call away. Services that do not use the loader import this package
directly — it depends only on the standard library.

- [Why](#why)
- [Value sources and resolution rules](#value-sources-and-resolution-rules)
- [Using it in a service](#using-it-in-a-service)
- [Deploying with systemd credentials](#deploying-with-systemd-credentials)
- [Deploying with Docker / Kubernetes](#deploying-with-docker--kubernetes)
- [Security notes and limits](#security-notes-and-limits)

## Why

Today secrets reach a service as env vars from `EnvironmentFile=` or as values
in its YAML config. Either way:

- they sit **in plain text on disk** (`/etc/default/webitel-*`, config files)
  and in every backup or image of that disk;
- an env var is readable from `/proc/<pid>/environ` and is **inherited by every
  child process**;
- anyone who can read the file can read the secret.

systemd (≥ 250) can store a secret encrypted with a key bound to the host
and/or its TPM2 chip, and hand the plaintext to the service only at runtime:

```
/etc/credstore.encrypted/webitel-im-account.postgres.dsn   encrypted, useless when copied elsewhere
            │  LoadCredentialEncrypted=   (decrypted by systemd at start)
            ▼
/run/credentials/webitel-im-account.service/postgres.dsn   plaintext, ramfs, mode 0400, owned by the
                                                          service user, invisible to other units,
                                                          removed when the service stops
```

The service finds it through the `$CREDENTIALS_DIRECTORY` env var — which is
exactly what this package reads.

## Value sources and resolution rules

A secret key's value is resolved once, in `Loader.Load` (or by
`Resolver.Value`):

| Configured value (flag / env / YAML) | Result |
|---|---|
| `cred:<name>` | contents of `$CREDENTIALS_DIRECTORY/<name>` |
| `file:<path>` | contents of `<path>` |
| any other non-empty literal | used as is (backward compatible) |
| empty / not set | conventional source, in this order: |
| | 1. `<ENV>_FILE` — file named by the env var, e.g. `POSTGRES_DSN_FILE=/run/secrets/pg_dsn` |
| | 2. systemd credential named exactly like the key, e.g. `postgres.dsn` |
| | 3. nothing — the key stays empty and the service's own validation reports it |

`<ENV>` is the loader's env name of the key: upper case, `.` and `-` replaced
by `_` (`service.secret_key` → `SERVICE_SECRET_KEY_FILE`).

Rules:

- **Conflicts fail loudly.** A key with a literal value *and* a conventional
  source (`POSTGRES_DSN` set and a `postgres.dsn` credential present) makes
  `Load` return `secret.ErrConflict`. This prevents a forgotten plain-text
  value from silently shadowing the credential that was meant to replace it.
  An explicit reference to the same source (`postgres.dsn: cred:postgres.dsn`)
  is fine.
- **Trailing `\n` / `\r\n` is trimmed** from file contents, so
  `echo "..." > file` works. Leading and inner whitespace is preserved.
- **An empty file is an error** (`secret.ErrEmpty`), as is a file over 1 MiB
  (systemd's own credential size limit).
- **`cred:` without systemd** (no `$CREDENTIALS_DIRECTORY`) is an error
  (`secret.ErrNoCredentialsDir`) — the reference can never resolve.
- **Error messages never contain secret contents**, only key names, credential
  names and paths.
- **Path keys** (`MarkSecretPath`, for settings that take a file path such as a
  TLS private key) resolve to the *path* of the source instead of its contents:
  `cred:tls.key` → `/run/credentials/<unit>/tls.key`.
- **Resolved once per process.** Resolved values are stored as viper
  overrides, so a `Loader.Watch` reload via `loader.Viper().Unmarshal(...)`
  keeps them; rotating a secret requires a restart. That matches systemd, which
  loads credentials only at service start.
- **Defaults never conflict.** A flag or `SetDefault` default is a fallback:
  a credential or `_FILE` replaces it, and it is used only when neither
  exists. Only a value set by a flag, an env var or the config file counts as
  configured.
- **Keys are case-insensitive**, as in viper. The conventional credential name
  is the lower-case key (`MarkSecret("Service.Secret_Key")` looks for
  `service.secret_key`).

## Using it in a service

### With `appconfig.Loader` (IM services)

Built-in secret keys are marked automatically for the enabled sections:

| Section | Secret key | Credential name | `_FILE` env |
|---|---|---|---|
| `Postgres` | `postgres.dsn` | `postgres.dsn` | `POSTGRES_DSN_FILE` |
| `Redis` | `redis.password` | `redis.password` | `REDIS_PASSWORD_FILE` |
| `Pubsub` | `pubsub.url` | `pubsub.url` | `PUBSUB_URL_FILE` |

Bumping `github.com/webitel/webitel-go-kit/appconfig` is enough for them.
Mark the service's own secrets before `Load`:

```go
loader := appconfig.NewLoader(appconfig.Sections{
    Log: true, Postgres: true, Redis: true, Consul: true, Pubsub: true,
})
loader.RegisterFlags(pflag.CommandLine)

pflag.String("service.secret_key", "", "32-byte AES key for token encryption (required)")
appconfig.RegisterGRPCConnFlags(pflag.CommandLine, "service.conn", true)

loader.
    MarkSecret("service.secret_key").                                  // value
    MarkSecretPath(appconfig.GRPCConnSecretPaths("service.conn")...)   // TLS key paths

pflag.Parse()

cfg := &Config{}
if err := loader.Load(pflag.CommandLine, cfg); err != nil {
    return nil, err // e.g. "appconfig: secret: value is set more than once: ..."
}
// cfg.Service.SecretKey holds the plaintext, whatever its source was.
```

Nothing else changes: the config struct still has plain `string` fields, and
existing `Watch` callbacks that call `loader.Viper().Unmarshal` keep working.

### Without the loader

```go
import "github.com/webitel/webitel-go-kit/appconfig/secret"

r := secret.FromEnv()

// key + its configured value (may be empty, a literal or a reference)
dsn, err := r.Value("postgres.dsn", os.Getenv("POSTGRES_DSN"))

// a single reference, without the conventional fallbacks
token, err := r.Resolve("cred:engine.token")

// a setting that expects a path
keyPath, err := r.PathValue("tls.key", cfg.TLSKeyPath)
```

`secret.New(dir, lookupEnv)` builds a resolver with an explicit credentials
directory and env lookup, which is handy in tests.

## Deploying with systemd credentials

Requires systemd ≥ 250 on the host (`systemctl --version`): Debian 12,
Ubuntu 24.04, RHEL 9 and newer. Ubuntu 22.04 (systemd 249) only has plain
`LoadCredential=` — no encryption; see [Older systemd](#older-systemd).

Example: move `POSTGRES_DSN` and `PUBSUB_URL` of `webitel-im-account` out of
`/etc/default/webitel-im-account`.

**1. Encrypt each secret** (as root). `systemd-ask-password` reads it without
echo and without leaving it in shell history; `-n` drops the trailing newline.

```sh
install -d -m 0700 /etc/credstore.encrypted

systemd-ask-password -n "postgres.dsn:" \
  | systemd-creds encrypt --name=postgres.dsn - \
      /etc/credstore.encrypted/webitel-im-account.postgres.dsn

systemd-ask-password -n "pubsub.url:" \
  | systemd-creds encrypt --name=pubsub.url - \
      /etc/credstore.encrypted/webitel-im-account.pubsub.url
```

`--name` must match the credential ID used in the unit. By default
(`--with-key=auto`) the secret is encrypted with the TPM2 chip when there is
one, plus the host key `/var/lib/systemd/credential.secret`. Use
`--with-key=host` on VMs without a virtual TPM, and `--with-key=tpm2` to bind
the secret to the physical machine only.

**2. Load them into the unit** with a drop-in, so the packaged unit file stays
untouched — `systemctl edit webitel-im-account`:

```ini
# /etc/systemd/system/webitel-im-account.service.d/credentials.conf
[Service]
LoadCredentialEncrypted=postgres.dsn:/etc/credstore.encrypted/webitel-im-account.postgres.dsn
LoadCredentialEncrypted=pubsub.url:/etc/credstore.encrypted/webitel-im-account.pubsub.url
```

The part before `:` is the credential name the service sees. Naming it after
the config key means no config change at all. Any other name works too when
the config references it explicitly: `postgres.dsn: cred:<name>`.

**3. Remove the plain-text values** — `POSTGRES_DSN=` and `PUBSUB_URL=` lines
from `/etc/default/webitel-im-account`, and the keys from the YAML config if
they are there. If you forget, the service refuses to start with a conflict
error naming the key, instead of silently using the old value.

**4. Restart and verify:**

```sh
systemctl daemon-reload
systemctl restart webitel-im-account
systemctl status webitel-im-account        # running, no "secret:" errors in the journal

# the plaintext is gone from disk and from the process environment
grep -c POSTGRES_DSN /etc/default/webitel-im-account                 # 0
tr '\0' '\n' < /proc/$(systemctl show -p MainPID --value webitel-im-account)/environ \
  | grep -c POSTGRES_DSN                                              # 0
```

#### Alternative: secret inside the drop-in

`SetCredentialEncrypted=` embeds the encrypted blob directly in the unit, with
no separate file:

```sh
systemd-ask-password -n "postgres.dsn:" \
  | systemd-creds encrypt --name=postgres.dsn -p - - \
  >> /etc/systemd/system/webitel-im-account.service.d/credentials.conf
```

`-p` prints a ready `SetCredentialEncrypted=postgres.dsn: ...` line (append it
under `[Service]`).

#### Many secrets per host: `ImportCredential=` (systemd ≥ 254)

`ImportCredential=webitel-im-account.*` loads every matching credential from
the credstore directories. The credential keeps its full name, so reference it
explicitly in config: `postgres.dsn: cred:webitel-im-account.postgres.dsn`.

#### TLS private keys

The mTLS keys from `service.conn.*` can be credentials as well, as long as the
service marks them with `MarkSecretPath`:

```ini
LoadCredentialEncrypted=service.conn.key:/etc/credstore.encrypted/webitel-im-account.tls.key
```

`service.conn.key` then resolves to `/run/credentials/webitel-im-account.service/service.conn.key`.
Certificates and CAs are not secret; keep them as plain paths.

#### Older systemd

On systemd 247–249 `LoadCredential=postgres.dsn:/etc/webitel/secrets/postgres.dsn`
still gives the service a private copy and keeps the secret out of the
environment, but the source file is plain text — keep it `0600 root:root`.

#### Local development without systemd

`systemd-run` gives a one-off process the same environment:

```sh
sudo systemd-run --pty --uid=$(id -u) \
  -p LoadCredentialEncrypted=postgres.dsn:/etc/credstore.encrypted/webitel-im-account.postgres.dsn \
  ./webitel-im-account server
```

On macOS just keep using `.env` (literal values) or `*_FILE` env vars.

To try it in a container that runs systemd as PID 1, make mounts shared first
(`mount --make-rshared /`): Docker's default private mount propagation hides
the credentials mount from the service and it sees an empty, unreadable
directory. Real hosts are not affected.

## Deploying with Docker / Kubernetes

Mount the secret as a file and point the `_FILE` env var at it:

```yaml
# docker compose
services:
  im-account:
    environment:
      POSTGRES_DSN_FILE: /run/secrets/postgres_dsn
    secrets: [postgres_dsn]
secrets:
  postgres_dsn:
    file: ./secrets/postgres_dsn   # gitignored
```

```yaml
# Kubernetes container spec
env:
  - name: POSTGRES_DSN_FILE
    value: /run/secrets/im/postgres.dsn
volumeMounts:
  - name: im-secrets
    mountPath: /run/secrets/im
    readOnly: true
```

Or reference the file from config: `postgres.dsn: file:/run/secrets/im/postgres.dsn`.

## Security notes and limits

- With `--with-key=host` the decryption key lives on the same machine
  (`/var/lib/systemd/credential.secret`, root only). That protects against
  leaked config files, backups and copied disks, not against root on a running
  host. `--with-key=tpm2` / `auto` on hardware with a TPM also binds the secret
  to that machine.
- Once loaded, the secret is a Go `string` in the service's memory, as before.
  The gain is that it is no longer stored in plain text, no longer in the
  process environment, and no longer inherited by child processes.
- The `cred:` / `file:` prefixes are only interpreted for keys marked as
  secrets; other config values are never touched. A literal secret that really
  starts with `cred:` or `file:` has to be supplied through a file.
