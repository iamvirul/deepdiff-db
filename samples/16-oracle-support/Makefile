.PHONY: up down seed diff gen-pack clean help wait-healthy

DEEPDIFFDB ?= deepdiffdb
CONFIG     := deepdiffdb.config.yaml

# sqlplus binary — override if not on PATH (e.g. from Oracle Instant Client)
SQLPLUS    ?= sqlplus

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-18s %s\n",$$1,$$2}'

up: ## Start both Oracle XE containers (takes ~60–90 s to become healthy)
	docker compose up -d
	@echo "Waiting for containers to become healthy (Oracle XE startup takes ~60–90 s)..."
	@$(MAKE) --no-print-directory wait-healthy

wait-healthy: ## Block until both containers pass their health checks
	@until [ "$$(docker compose ps --format json | python3 -c "import sys,json; data=sys.stdin.read().strip(); rows=[json.loads(l) for l in data.splitlines() if l.strip()]; print(sum(1 for r in rows if r.get('Health')=='healthy'))" 2>/dev/null)" = "2" ]; do \
	    echo "  still waiting..."; sleep 10; \
	done
	@echo "Both Oracle containers are healthy."

seed: up ## Initialise prod and dev schemas (drops and recreates tables)
	docker cp init-scripts/01-prod-schema.sql $$(docker compose ps -q oracle-prod):/tmp/01-prod-schema.sql
	docker exec $$(docker compose ps -q oracle-prod) bash -c \
	    'sqlplus -S system/"$$ORACLE_PASSWORD"@localhost:1521/XEPDB1 @/tmp/01-prod-schema.sql'
	docker cp init-scripts/02-dev-schema.sql $$(docker compose ps -q oracle-dev):/tmp/02-dev-schema.sql
	docker exec $$(docker compose ps -q oracle-dev) bash -c \
	    'sqlplus -S system/"$$ORACLE_PASSWORD"@localhost:1521/XEPDB1 @/tmp/02-dev-schema.sql'
	@echo "Databases seeded."

diff: ## Run full diff (schema + content)
	$(DEEPDIFFDB) diff --config $(CONFIG)

gen-pack: ## Generate a migration pack
	$(DEEPDIFFDB) gen-pack --config $(CONFIG)

down: ## Stop and remove containers and volumes
	docker compose down -v

clean: down ## Remove containers, volumes, and generated output
	rm -rf diff-output/*.json diff-output/*.txt diff-output/*.sql diff-output/*.zip diff-output/*.html
