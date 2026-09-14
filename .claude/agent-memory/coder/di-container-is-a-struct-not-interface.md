---
name: di-container-is-a-struct-not-interface
description: sarulabs/di v2 Container is a STRUCT whose zero value nil-derefs inside SubContainer() — so DiContainerMiddleware can panic, which decides where recover must be mounted
metadata:
  type: reference
---

`di.Container` (sarulabs/di/v2 v2.5.2) is a **struct**, not an interface:

```go
type Container struct {
    core      *containerCore
    builtList []int
}
```

Two consequences that are invisible from reading service code.

**1. You cannot fake it.** No `type fakeContainer struct{ di.Container }` with an
overridden method, and no stub implementation — any test that needs a container
must build a real one with `di.NewBuilder()`. That is cheap (`builder.Add(di.Def{
Name:…, Scope: di.Request, Build:…, Close:…})` then `builder.Build()`), so reach
for it rather than trying to mock.

**2. The zero value nil-panics.** `Container.SubContainer()` dereferences
`ctn.core.scopeLevel` on its first line, so `di.Container{}` SIGSEGVs there. This
makes `DiContainerMiddleware(app)` a **panicking** middleware whenever `iapp.App`
has not been built — exactly the state `iapp.App` is in until `app.Init()` runs,
i.e. one boot-order regression away.

**Why it matters: it settles the recover-vs-DI mount order**, which otherwise
looks like a toss-up. Two properties compete:

- *release the request container on every path* — holds in EITHER order, because
  the release is a `defer` and Go runs deferreds during a panic unwind too. Not a
  tiebreaker, despite being the thing people worry about.
- *recover the panic* — does NOT hold in either order. Only recover-OUTSIDE-DI
  catches a panic raised by the DI middleware itself, and fasthttp does not
  recover panics, so uncaught means the process dies.

So: `cors -> access log -> recover -> di container -> routes`. Keep recover
**below** the access log though — fiberzerolog logs after `c.Next()` with no defer
of its own, so a panic unwinding past it is never logged; with recover inside, the
panic becomes a 500 first and the request still shows up.

⚠️ The platform preset and gardener/isme/medioa2 all mount recover INSIDE the DI
middleware, and the preset comment justifies it with "the release is a defer, so
it runs while a panic unwinds (mount recover INSIDE this one)" — a non sequitur,
since the defer runs either way. tomatime PR #22 deviates deliberately. If you
touch that ordering in another service, this is the argument.

**How to test it:** `mountMiddlewares(app, di.Container{})` → assert 500. The
mutation (swap the order back) does not fail cleanly, it **SIGSEGVs the test
binary** — which is the correct red, because it is the production consequence
reproduced. Extract the middleware registration into a method so the test
exercises the order the server actually mounts.

Related: [[feedback-prove-regression-tests]], [[fiber-ip-and-buffer-gotchas]].
