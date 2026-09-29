# pkg/desire/store

Backend implementations of the `desire.SpecStore` / `desire.StatusStore` contracts defined in
`pkg/desire` (see `pkg/desire/CLAUDE.md`).

- **memory** — single-process, mutex-guarded map; used in unit tests and envtest.
- **redis** — one JSON `resourceRecord` per desire, updated with WATCH/MULTI/EXEC CAS. `Create*`
  uses multi-key WATCH to enforce shared-owner and apply/delete rules atomically.
- **generation compatibility** — Redis records written before `Generation` existed decode with a
  baseline of 1; their old conditions have `ObservedGeneration` 0 and remain stale until the
  controller reconciles. `Version` cannot reconstruct historical spec generations because status
  writes also advanced it. The next write persists the normalized generation.
- **conformance** — `RunSpecStoreSuite` and `RunStatusStoreSuite` run against both backends. Put
  shared backend behavior tests here unless the behavior is backend-specific.
- **spec comparison** — both backends use the same JSON-value comparison when deciding whether an
  accepted Apply spec update advances `Generation`. The CAS `Version` still advances on every
  accepted update; JSON formatting and object key order alone do not make a new generation.
