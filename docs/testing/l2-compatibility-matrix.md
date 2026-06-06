# L2 Compatibility Matrix

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

Compatibility should be recorded against adapter family, selected contract packs, test profiles, target release level, and observed behavior notes. The golden sample registry `.agent/registry/l2-golden-samples.yaml` lists initial standards-only expectations for redisx, postgresx, kafkax, natsx, ossx, clickhousex, and taosx.

## Matrix dimensions

- Adapter family and module identity.
- Selected packs such as `common`, `kv`, `ttl`, `sql`, `pubsub`, or `objectstore`.
- Required profiles for the requested release level.
- Evidence paths and report statuses.
- Compatibility notes for version drift, semantic gaps, and migration risks.

## Use in reviews

The matrix helps reviewers compare adapters without provider-specific implementation details. A compatibility claim must link to evidence; unsupported capabilities should be explicit rather than hidden by omitted rows.

## Evolution

New pack candidates and behavior gaps should enter the registry backlog first. Once promoted to first-class pack semantics, downstream repositories update manifests, regenerate evidence, and refresh compatibility matrices.

Evidence paths for release claims should remain under `.agent/evidence/l2` unless resiliencx changes the schema.
