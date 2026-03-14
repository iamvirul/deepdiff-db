.PHONY: up down seed diff diff-schema diff-content gen-pack clean help wait-healthy

DEEPDIFFDB ?= deepdiffdb
CONFIG     := deepdiffdb.config.yaml

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-18s %s\n",$$1,$$2}'

up: ## Start both SQL Server containers
	docker compose up -d
	@echo "Waiting for containers to become healthy..."
	@$(MAKE) --no-print-directory wait-healthy

wait-healthy: ## Block until both containers pass their health checks
	@until [ "$$(docker compose ps --format json | python3 -c "import sys,json; data=sys.stdin.read().strip(); rows=[json.loads(l) for l in data.splitlines() if l.strip()]; print(sum(1 for r in rows if r.get('Health')=='healthy'))" 2>/dev/null)" = "2" ]; do \
	    echo "  still waiting..."; sleep 5; \
	done
	@echo "Both containers are healthy."

seed: up ## Initialise prod and dev databases (drops and recreates tables)
	docker compose exec -T mssql-prod /opt/mssql-tools18/bin/sqlcmd \
	    -S localhost -U sa -P 'StrongP@ss1word!' -C \
	    -i /dev/stdin < init-scripts/01-prod-schema.sql
	docker compose exec -T mssql-dev /opt/mssql-tools18/bin/sqlcmd \
	    -S localhost -U sa -P 'StrongP@ss1word!' -C \
	    -i /dev/stdin < init-scripts/02-dev-schema.sql
	@echo "Databases seeded."

diff: ## Run full diff (schema + content)
	$(DEEPDIFFDB) diff --config $(CONFIG)

diff-schema: ## Run schema-only diff
	$(DEEPDIFFDB) diff --config $(CONFIG) --skip-content

diff-content: ## Run content-only diff
	$(DEEPDIFFDB) diff --config $(CONFIG) --skip-schema

gen-pack: ## Generate a migration pack
	$(DEEPDIFFDB) gen-pack --config $(CONFIG)

down: ## Stop and remove containers and volumes
	docker compose down -v

clean: down ## Remove containers, volumes, and generated output
	rm -rf diff-output/*.json diff-output/*.txt diff-output/*.sql diff-output/*.zip diff-output/*.html
