#!/usr/bin/env bash
# demo.sh — end-to-end walkthrough of DeepDiff DB git-like versioning
# Run from the sample root:  bash scripts/demo.sh
set -euo pipefail

SAMPLE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SAMPLE_DIR"

# ── colour helpers ──────────────────────────────────────────────────────────
BOLD='\033[1m'; CYAN='\033[0;36m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'
header()  { echo -e "\n${BOLD}${CYAN}==> $*${RESET}"; }
step()    { echo -e "${GREEN}  $*${RESET}"; }
note()    { echo -e "${YELLOW}  NOTE: $*${RESET}"; }

# ── resolve binary ──────────────────────────────────────────────────────────
if command -v deepdiffdb &>/dev/null; then
    BIN="deepdiffdb"
elif [ -f "../../deepdiffdb" ]; then
    BIN="../../deepdiffdb"
else
    echo "deepdiffdb binary not found. Build it with: go build -o deepdiffdb ./cmd/deepdiffdb/" >&2
    exit 1
fi

MYSQL_PROD="docker exec -i vcs_prod_db mysql -uroot -prootpassword shop"
MYSQL_DEV="docker exec -i vcs_dev_db  mysql -uroot -prootpassword shop"

# ── step 0: start containers ────────────────────────────────────────────────
header "Step 0 — Start databases"
docker-compose up -d
step "Waiting for MySQL to be ready..."
for i in $(seq 1 30); do
    if docker exec vcs_prod_db mysqladmin ping -prootpassword --silent 2>/dev/null \
    && docker exec vcs_dev_db  mysqladmin ping -prootpassword --silent 2>/dev/null; then
        step "Both databases ready."
        break
    fi
    sleep 2
    if [ "$i" -eq 30 ]; then
        echo "Databases did not become ready in time." >&2
        exit 1
    fi
done
sleep 3   # extra grace period for init scripts

# ── step 1: init version repository ────────────────────────────────────────
header "Step 1 — Initialise version repository"
"$BIN" version init
note "A .deepdiffdb/ directory now exists in this sample folder."

# ── step 2: commit V1 (prod == dev, clean baseline) ─────────────────────────
header "Step 2 — Commit V1: clean baseline"
step "Both databases have identical schemas at this point."
"$BIN" version commit \
    --config deepdiffdb.config.yaml \
    --message "V1: baseline e-commerce schema (categories, products, orders)" \
    --author  "Alice"
HASH_V1=$("$BIN" version log --limit 1 | awk '/^commit/ {print $2; exit}')
step "Committed as ${HASH_V1:0:8}"

# ── step 3: apply V2 migration to dev ───────────────────────────────────────
header "Step 3 — Apply V2 schema changes to dev database"
step "Adding category_id FK to products and customer_email to orders..."
$MYSQL_DEV < migrations/v2_add_category_link_and_customer.sql
step "V2 migration applied to dev."

# ── step 4: commit V2 ───────────────────────────────────────────────────────
header "Step 4 — Commit V2: category link + customer email"
"$BIN" version commit \
    --config deepdiffdb.config.yaml \
    --message "V2: link products to categories, capture customer_email on orders" \
    --author  "Alice"
HASH_V2=$("$BIN" version log --limit 1 | awk '/^commit/ {print $2; exit}')
step "Committed as ${HASH_V2:0:8}"

# ── step 5: apply V3 migration to dev ───────────────────────────────────────
header "Step 5 — Apply V3 schema changes to dev database"
step "Adding reviews table and avg_rating column to products..."
$MYSQL_DEV < migrations/v3_add_reviews.sql
step "V3 migration applied to dev."

# ── step 6: commit V3 ───────────────────────────────────────────────────────
header "Step 6 — Commit V3: reviews + avg_rating"
"$BIN" version commit \
    --config deepdiffdb.config.yaml \
    --message "V3: add reviews table and avg_rating denorm column on products" \
    --author  "Bob"
HASH_V3=$("$BIN" version log --limit 1 | awk '/^commit/ {print $2; exit}')
step "Committed as ${HASH_V3:0:8}"

# ── step 7: view history ─────────────────────────────────────────────────────
header "Step 7 — Version log (full history)"
"$BIN" version log

# ── step 8: compare V1 → V3 ─────────────────────────────────────────────────
header "Step 8 — Schema evolution: V1 → V3"
"$BIN" version diff "$HASH_V1" "$HASH_V3"

# ── step 9: compare V2 → V3 ─────────────────────────────────────────────────
header "Step 9 — Schema evolution: V2 → V3"
"$BIN" version diff "$HASH_V2" "$HASH_V3"

# ── step 10: generate rollback SQL for V3 ────────────────────────────────────
header "Step 10 — Generate rollback SQL for V3"
mkdir -p diff-output
"$BIN" version rollback --out diff-output/rollback_v3.sql "$HASH_V3"
step "Rollback SQL written to diff-output/rollback_v3.sql"
echo ""
echo "Preview:"
head -30 diff-output/rollback_v3.sql

# ── step 11: generate rollback SQL for V2 ────────────────────────────────────
header "Step 11 — Generate rollback SQL for V2"
"$BIN" version rollback --out diff-output/rollback_v2.sql "$HASH_V2"
step "Rollback SQL written to diff-output/rollback_v2.sql"

echo ""
echo -e "${BOLD}${GREEN}Demo complete.${RESET}"
echo ""
echo "Artifacts:"
echo "  .deepdiffdb/objects/   — commit objects (zlib-compressed, Git-style fanout)"
echo "  diff-output/rollback_v3.sql — SQL to undo V3 changes"
echo "  diff-output/rollback_v2.sql — SQL to undo V2 changes"
echo ""
echo "Clean up:  docker-compose down -v && rm -rf .deepdiffdb diff-output/*.sql"
