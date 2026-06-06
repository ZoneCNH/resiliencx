# L2 Capability Manifest

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

The manifest schema is `.agent/schemas/l2-capabilities.schema.json`; the downstream template is `templates/l2/.agent/l2-capabilities.yaml`. The template is intentionally local and provider-neutral so new adapters can start by proving shape before wiring testkitx.

## Required sections

- `schema_version` identifies the standards schema version.
- `layer` must be `L2`.
- `adapter` records name, module, family, and owners.
- `capabilities` declares each adapter capability with family and status.
- `contract_packs` maps capabilities to resiliencx registry packs.
- `evidence` records required profiles, output directory, and report paths.

## Invariants

Every selected pack must exist in `.agent/registry/l2-contract-packs.yaml`. Evidence paths should stay under `.agent/evidence/l2`. Manifests should describe adapter intent only; live connection details, runtime credential loading, and runner wiring belong in the adopting repository and should not be committed into resiliencx artifacts.

## Local checks

Run `make l2-capability-check` in the template to confirm the manifest exists. Downstream repositories should then validate the manifest against the schema before invoking testkitx or xlibgate.
