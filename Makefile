.PHONY: test-examples

test-examples:
	@find examples -name main.go -print | while read -r file; do \
		dir="$$(dirname "$$file")"; \
		echo "[run] $$dir"; \
		(cd "$$dir" && go run .); \
	done
