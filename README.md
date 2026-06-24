# settings — shared config store for togo

A togo plugin that gives every other plugin a **shared, typed config store**. Values
are JSON, addressed by `(scope, key)` — use `global`, a plugin name, or a tenant id —
and persisted in the kernel database.

```bash
togo install togo-framework/settings
```

## Go API

```go
s := settings.FromKernel(k)               // *settings.Store from the kernel
_ = s.Set(ctx, settings.ScopeGlobal, "site.title", "ToGO")
title, ok, _ := settings.Get[string](ctx, s, settings.ScopeGlobal, "site.title")
all, _ := s.All(ctx, "billing")           // every key in a scope
_ = s.Delete(ctx, "billing", "plan")
```

## REST API

| Method | Path | Body |
|---|---|---|
| `GET`    | `/api/settings/{scope}`        | — → `{key: value}` |
| `GET`    | `/api/settings/{scope}/{key}`  | — → value |
| `PUT`    | `/api/settings/{scope}/{key}`  | any JSON |
| `DELETE` | `/api/settings/{scope}/{key}`  | — |

## Data model

`settings(scope text, skey text, value text json, PRIMARY KEY (scope, skey))` — created
on boot. Cross-driver TEXT columns; upsert via `ON CONFLICT`.

MIT
