# L2 Release Gate

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

Release levels are defined in `.agent/registry/l2-release-levels.yaml` and may not be redefined downstream. L2-T3 is the first release-allowed level; L2-T4 is factory-grade.

## Level expectations

- `L2-T0` proves skeleton readiness only.
- `L2-T1` requires unit and contract profiles.
- `L2-T2` adds integration evidence.
- `L2-T3` adds chaos, benchmark, and adoption evidence and allows release.
- `L2-T4` adds retrospective evidence and allows factory-grade claims.

## Gate behavior

Gates fail closed when required profiles, evidence paths, selected pack reports, or compatibility notes are missing. xlibgate adjudicates the evidence against this registry; resiliencx only defines the release semantics and template targets.

## Template targets

`make l2-release-readiness` checks the local standards shape and points downstream to xlibgate. The compatibility aliases `l2-release-readiness-check`, `l2-manifest-check`, and `l2-evidence-check` remain available for existing workflows.

Evidence paths for release claims should remain under `.agent/evidence/l2` unless resiliencx changes the schema.
