BIN = ../../deepdiffdb

.PHONY: help up wait-healthy seed diff row-diff html down clean

help:
	@echo "Sample 18: Cross-Engine Row Diff & Migration Verification"
	@echo ""
	@echo "Targets:"
	@echo "  make up            - Start PostgreSQL 16 and Oracle XE 21c containers"
	@echo "  make wait-healthy  - Wait for both containers to report healthy"
	@echo "  make seed          - Initialize schemas and seed drifted data"
	@echo "  make diff          - Run full cross-engine diff (schema + data)"
	@echo "  make row-diff      - Inspect column-level differences in terminal"
	@echo "  make html          - Generate and open interactive HTML report"
	@echo "  make down          - Stop and remove Docker containers"
	@echo "  make clean         - Remove diff-output artifacts"

up:
	docker compose up -d

wait-healthy:
	@echo "Waiting for PostgreSQL to be healthy..."
	@until [ "$$(docker inspect --format='{{.State.Health.Status}}' sample18-postgres 2>/dev/null)" = "healthy" ]; do sleep 2; done
	@echo "PostgreSQL is ready."
	@echo "Waiting for Oracle to be healthy (takes ~45-90s on initial start)..."
	@until [ "$$(docker inspect --format='{{.State.Health.Status}}' sample18-oracle 2>/dev/null)" = "healthy" ]; do sleep 5; done
	@echo "Oracle XE is ready."

seed:
	@echo "Seeding PostgreSQL..."
	docker exec -i sample18-postgres psql -U postgres -d appdb < init-scripts/01-postgres-init.sql
	@echo "Seeding Oracle XE..."
	docker exec -i sample18-oracle sqlplus -S appuser/secret@XEPDB1 < init-scripts/02-oracle-init.sql
	@echo "Seeding complete."

diff:
	$(BIN) diff --config deepdiffdb.config.yaml

row-diff:
	$(BIN) row-diff --config deepdiffdb.config.yaml --status all

html:
	$(BIN) diff --config deepdiffdb.config.yaml --html --row-diff
	@echo "Opening diff-output/report.html..."
	@open diff-output/report.html 2>/dev/null || xdg-open diff-output/report.html 2>/dev/null || echo "Report available at diff-output/report.html"

down:
	docker compose down -v

clean:
	rm -rf diff-output
