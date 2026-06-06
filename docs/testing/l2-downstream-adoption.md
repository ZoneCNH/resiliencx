# L2 Downstream Adoption

resiliencx is the standards source. It defines artifact shape, required evidence, and release semantics only. testkitx owns executable contract libraries, and xlibgate owns release adjudication. This repository must remain provider-neutral: no provider connections, no credentials loading, and no Contract Runner implementation live here.

Adopting repositories consume resiliencx templates and registries without redefining release levels. They may add provider-specific services, credentials loading, concrete compose services, and Contract Runner wiring only in their own repository.

## Adoption checklist

- Keep `.agent/l2-capabilities.yaml` schema-valid.
- Select only registry-defined contract packs unless a backlog item has been promoted.
- Preserve evidence under `.agent/evidence/l2`.
- Keep resiliencx template aliases available when local Makefiles wrap them.
- Publish compliance, compatibility, and release-readiness reports with each release claim.

## Template behavior

The included template starts with local shape checks and provider-neutral placeholders. The compose file uses a placeholder profile and no network access by default, so adopting repositories can prove paths before adding services.

## Upgrade path

When resiliencx updates schemas or release levels, downstream repositories should update manifests first, rerun shape checks, then rerun executable testkitx profiles and xlibgate adjudication.
