.PHONY: verify-contracts verify-contracts-reproducible

# The Go binary embeds x/erc721/contracts/compiled_contracts/ERC721Uptick.json
# (go:embed cannot reference files outside the package subtree), so the same
# artifact exists twice by necessity. This check pins the embedded copy to the
# canonical root artifact so the two can never drift (L-12).
EMBEDDED_DIR := ../x/erc721/contracts/compiled_contracts

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
	@if [ -f "$(EMBEDDED_DIR)/ERC721Uptick.json" ]; then \
		if ! cmp -s compiled_contracts/ERC721Uptick.json "$(EMBEDDED_DIR)/ERC721Uptick.json"; then \
			echo "MISMATCH: embedded x/erc721 copy differs from canonical compiled_contracts/ERC721Uptick.json"; \
			exit 1; \
		fi; \
		echo "ok: embedded x/erc721 copy matches canonical artifact"; \
	else \
		echo "MISSING: $(EMBEDDED_DIR)/ERC721Uptick.json (go:embed artifact)"; \
		exit 1; \
	fi

# Recompile with the pinned solc/OZ and compare all creation/runtime bytecode
# against the committed reference hashes in refs/ (L-11 reproducibility gate).
verify-contracts-reproducible:
	@command -v solc >/dev/null 2>&1 || { echo "solc not installed"; exit 1; }
	@npm ci >/dev/null 2>&1 || { echo "npm ci failed"; exit 1; }
	@./verify_contracts.sh
	@rm -rf node_modules
