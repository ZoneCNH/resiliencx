# L2 Contract Pack Registry

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

The registry is `.agent/registry/l2-contract-packs.yaml` and is validated by `.agent/schemas/l2-contract-packs.schema.json`. Pack entries define family, title, required profiles, required evidence, and capability names.

## Pack fields

- `family` groups capabilities into common, key-value, relational, messaging, streaming, storage, analytics, or time-series domains.
- `profiles` names the evidence stages needed to prove the pack.
- `required_evidence` names report classes expected from downstream execution.
- `capabilities` lists semantic operations covered by the pack.

## Backlog discipline

`extension_backlog` records standards candidates that are not yet first-class pack definitions or still need downstream evidence maturity. Items such as `ttl` may appear there when leader QA wants explicit tracking even if an initial pack exists. Backlog entries are planning markers only; they are not executable tests.

## Ownership boundary

resiliencx may add, rename, or retire pack definitions. testkitx implements the executable checks. Downstream adapters select packs and publish evidence; they do not redefine pack semantics locally.

Evidence paths for release claims should remain under `.agent/evidence/l2` unless resiliencx changes the schema.
