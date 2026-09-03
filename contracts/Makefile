.PHONY: verify-contracts

# Leave the exact solc/OZ pins to package.json and the README; these
# verification targets are intended to run in CI with the pinned toolchain.
verify-contracts:
	@echo "verify-contracts: (CI job) rebuild with solc 0.8.28 + OZ 4.9.6, then"
	@echo "compare each compiled_contracts/*.json against checksums.txt."
	@for f in compiled_contracts/*.json; do \
		h=$$(shasum -a 256 "$$f" | awk '{print $$1}'); \
		p=$$(awk -v f="$$f" '$$2 == f {print $$1}' checksums.txt); \
		if [ "$$h" != "$$p" ]; then echo "MISMATCH: $$f"; exit 1; fi; \
		echo "ok: $$f"; \
	done
