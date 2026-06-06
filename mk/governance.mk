# Reusable governance fragment copied into rendered downstream repositories.
GOALCLI ?= go run ./cmd/goalcli

.PHONY: require-gowork-off
require-gowork-off:
	@if [ "$${GOWORK:-}" != "off" ]; then \
		echo "GOWORK=off is required for adoption-check"; \
		exit 1; \
	fi

.PHONY: adoption-check
adoption-check: require-gowork-off
	$(GOALCLI) adoption-check --verify
