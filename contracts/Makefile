.PHONY: verify-contracts verify-contracts-reproducible

# Verify the committed artifacts' integrity against checksums.txt.
verify-contracts:
	@echo "verify-contracts: (CI job) rebuild with solc 0.8.28 + OZ 4.9.6, then"
	@echo "compare each compiled_contracts/*.json against checksums.txt."
	@for f in compiled_contracts/*.json; do \
		h=$$(shasum -a 256 "$$f" | awk '{print $$1}'); \
		p=$$(awk -v f="$$f" '$$2 == f {print $$1}' checksums.txt); \
		if [ "$$h" != "$$p" ]; then echo "MISMATCH: $$f"; exit 1; fi; \
		echo "ok: $$f"; \
	done

# Recompile with the pinned solc/OZ and compare all creation/runtime bytecode
# against the committed reference hashes in refs/ (L-11 reproducibility gate).
verify-contracts-reproducible:
	@command -v solc >/dev/null 2>&1 || { echo "solc not installed"; exit 1; }
	@npm ci >/dev/null 2>&1 || { echo "npm ci failed"; exit 1; }
	@./verify_contracts.sh
	@rm -rf node_modules
