# L2 Compliance Matrix

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

The compliance matrix maps requirement, contract pack, profile, evidence path, status, and release-level impact. Its schema is `.agent/schemas/l2-compliance-matrix.schema.json`.

## Required mapping

Each row should identify the adapter capability, the registry pack that owns the semantic expectation, the profile that proved it, and the evidence file that backs the status. Statuses are `pass`, `fail`, `missing`, or `not_applicable`.

## Matrix rules

- A capability declared in the manifest should appear in the matrix.
- A selected contract pack should have at least one evidence-backed row.
- `missing` and `fail` rows must block the affected release level.
- `not_applicable` must be justified by adapter family or explicit unsupported capability status.

## Review use

xlibgate and human reviewers should be able to reconstruct the release decision from the matrix plus linked evidence without reading downstream implementation code.

Evidence paths for release claims should remain under `.agent/evidence/l2` unless resiliencx changes the schema.
