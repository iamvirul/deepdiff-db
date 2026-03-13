.PHONY: seed seed-small diff diff-fast diff-sequential gen-pack clean help

DEEPDIFFDB ?= deepdiffdb
CONFIG     := deepdiffdb.config.yaml

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-18s %s\n",$$1,$$2}'

seed: ## Generate full databases (~800k rows; ~60s)
	cd seed && go run . && mv prod.db dev.db ../
	@echo "Databases ready: prod.db dev.db"

seed-small: ## Generate small databases (10k/2k/5k rows; ~5s)
	cd seed && go run . --small && mv prod.db dev.db ../
	@echo "Databases ready: prod.db dev.db (small)"

diff: ## Diff with config defaults (batch=10000, parallel=2)
	$(DEEPDIFFDB) diff --config $(CONFIG)

diff-fast: ## Diff overriding flags: batch=5000, parallel=4
	$(DEEPDIFFDB) diff --config $(CONFIG) --batch-size 5000 --parallel 4

diff-sequential: ## Diff without batching or parallelism (baseline)
	$(DEEPDIFFDB) diff --config $(CONFIG) --batch-size 0 --parallel 1

gen-pack: ## Generate a migration pack (uses config performance settings)
	$(DEEPDIFFDB) gen-pack --config $(CONFIG)

clean: ## Remove generated databases and output
	rm -f prod.db dev.db
	rm -rf diff-output/*.json diff-output/*.txt diff-output/*.sql diff-output/*.zip
