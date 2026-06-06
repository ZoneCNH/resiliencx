# L2 Evidence Standard

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

Evidence must be reproducible, machine-readable where possible, and linked to manifest capabilities, selected contract packs, required profiles, and release-level decisions. The canonical downstream output root is `.agent/evidence/l2`.

## Minimum evidence set

- Contract report for each selected pack.
- Compliance matrix mapping requirements, packs, profiles, evidence paths, and status.
- Integration, chaos, benchmark, and adoption reports when required by the requested release level.
- Release-readiness summary produced for xlibgate adjudication.

## Evidence quality

Evidence should name the manifest version, adapter module, selected packs, command or workflow that produced the report, timestamp, and pass/fail status. A missing file or empty placeholder is not passing evidence. Standards templates may include `.gitkeep` files to establish directories, but downstream release claims must include real reports.

## Failure handling

When a profile fails, preserve the failing evidence and record the blocker. Do not remove the profile, lower the target release level without review, or add provider-specific bypasses to resiliencx.
