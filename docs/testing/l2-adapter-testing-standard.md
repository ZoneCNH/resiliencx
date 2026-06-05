# L2 Adapter Testing Standard

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

## Required downstream flow

1. Copy `templates/l2` into the L2 adapter repository.
2. Complete `.agent/l2-capabilities.yaml` with adapter metadata, capability declarations, selected contract packs, and evidence paths.
3. Run the template shape gates first: `make l2-capability-check l2-evidence`.
4. Wire executable testkitx packs downstream and keep their reports under `.agent/evidence/l2`.
5. Run profile stages in order: `l2-contract`, `l2-integration`, `l2-chaos`, `l2-benchmark`, and `l2-adoption`.
6. Submit evidence to xlibgate for release-readiness adjudication without weakening resiliencx release levels.

## Profile semantics

- `skeleton` proves the manifest and evidence directories exist.
- `unit` proves local adapter behavior without live provider dependencies.
- `contract` proves declared packs such as `common`, `kv`, and `ttl` against testkitx.
- `integration`, `chaos`, and `benchmark` prove downstream service behavior and must be recorded as evidence, not encoded in this standards repository.
- `adoption` and `retrospective` prove rollout safety, compatibility notes, and post-release learnings.

## Pass criteria

A downstream adapter is L2-ready only when every declared capability maps to a registry pack, every required profile has machine-readable evidence, and xlibgate confirms the requested release level. Missing evidence is a blocker, not a reason to lower the standard.
