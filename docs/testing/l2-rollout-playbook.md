# L2 Rollout Playbook

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

## Sequence

1. Copy `templates/l2` into the L2 repository.
2. Fill adapter metadata, declared capabilities, selected contract packs, and evidence report paths.
3. Run `make l2-capability-check l2-evidence` to prove local shape.
4. Wire testkitx in the downstream repository and generate contract reports.
5. Add integration, chaos, benchmark, and adoption evidence as the target release level requires.
6. Run `make l2-release-readiness` and submit the evidence bundle to xlibgate.
7. Record blockers instead of weakening release criteria.

## Rollout controls

Start at `L2-T0` until the manifest and evidence directory are stable. Promote one release level at a time, preserving failed evidence and review notes. Keep compatibility notes for adapters that share families or pack selections so regressions can be compared across downstream repositories.

## Stop conditions

Stop rollout when selected packs lack executable testkitx coverage, required reports are missing, or provider-specific behavior cannot be represented by existing pack semantics. Escalate the gap through the resiliencx backlog rather than adding local exceptions.

Evidence paths for release claims should remain under `.agent/evidence/l2` unless resiliencx changes the schema.
