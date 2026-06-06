L2_MANIFEST ?= .agent/l2-capabilities.yaml
L2_EVIDENCE_DIR ?= .agent/evidence/l2

.PHONY: \
	l2-capability-check \
	l2-contract \
	l2-integration \
	l2-chaos \
	l2-benchmark \
	l2-adoption \
	l2-evidence \
	l2-release-readiness \
	l2-manifest-check \
	l2-contract-placeholder \
	l2-evidence-check \
	l2-release-readiness-check

l2-capability-check:
	@test -f $(L2_MANIFEST)
	@echo "L2 capability manifest present: $(L2_MANIFEST)"

l2-contract: l2-capability-check
	@echo "Standards-only contract stage: declare packs here; execute testkitx downstream"

l2-integration: l2-capability-check
	@echo "Standards-only integration stage: provider-neutral placeholder, downstream wires services"

l2-chaos: l2-capability-check
	@echo "Standards-only chaos stage: declare fault evidence expectations downstream"

l2-benchmark: l2-capability-check
	@echo "Standards-only benchmark stage: declare performance evidence expectations downstream"

l2-adoption: l2-capability-check
	@echo "Standards-only adoption stage: confirm rollout evidence and compatibility matrix downstream"

l2-evidence:
	@test -d $(L2_EVIDENCE_DIR)
	@echo "L2 evidence directory present: $(L2_EVIDENCE_DIR)"

l2-release-readiness: l2-capability-check l2-evidence
	@echo "Run xlibgate release-readiness adjudication downstream"

l2-manifest-check: l2-capability-check

l2-contract-placeholder: l2-contract

l2-evidence-check: l2-evidence

l2-release-readiness-check: l2-release-readiness
