<!-- togo-header -->
<div align="center">
  <img src=".github/assets/togo-mark.svg" alt="togo" height="64" />
  <h1>togo-framework/settings</h1>
  <p>
    <a href="https://to-go.dev/marketplace"><img src="https://img.shields.io/badge/marketplace-to--go.dev-1FC7DC" alt="marketplace" /></a>
    <a href="https://pkg.go.dev/github.com/togo-framework/settings"><img src="https://pkg.go.dev/badge/github.com/togo-framework/settings.svg" alt="pkg.go.dev" /></a>
    <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT" />
  </p>
  <p><strong>Part of the <a href="https://to-go.dev">togo</a> framework.</strong></p>
</div>

## Install

```bash
togo install togo-framework/settings
```

<!-- /togo-header -->

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

<!-- togo-sponsors -->
---

<div align="center">
  <h3>Premium sponsors</h3>
  <p>
    <a href="https://id8media.com"><strong>ID8 Media</strong></a> &nbsp;·&nbsp;
    <a href="https://one-studio.co"><strong>One Studio</strong></a>
  </p>
  <p><sub>Support togo — <a href="https://github.com/sponsors/fadymondy">become a sponsor</a>.</sub></p>
</div>
<!-- /togo-sponsors -->
