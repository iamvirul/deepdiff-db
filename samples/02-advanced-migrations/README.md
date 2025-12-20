# Advanced Migrations Sample

This sample demonstrates how to use all the commands of `deepdiffdb` to detect schema and data drift, generate a migration pack, and apply it to a database.

## 1. Start the Databases

First, start the three MySQL databases using Docker Compose:

```bash
docker-compose up -d
```

This will start three MySQL instances:
- `db_source`: on `localhost:3310`
- `db_target`: on `localhost:3311`
- `db_apply`: on `localhost:3312`

The root password for all databases is `rootpassword`.

## 2. Validate Configuration

The `check` command validates the configuration file and shows a quick summary of the databases.

```bash
deepdiffdb check -config deepdiffdb.config.yaml
```

## 3. Detect Schema Drift

The `schema-diff` command detects differences in the database schemas.

```bash
deepdiffdb schema-diff -config deepdiffdb.config.yaml
```

This will generate a schema diff report in the `diff-output` directory.

## 4. Full Diff (Schema + Data)

The `diff` command performs a full diff, including both schema and data.

```bash
deepdiffdb diff -config deepdiffdb.config.yaml
```

This will generate a full diff report in the `diff-output` directory.

## 5. Generate SQL Migration Pack

The `gen-pack` command generates a SQL migration pack to transform the source database into the target database.

```bash
deepdiffdb gen-pack -config deepdiffdb.config.yaml
```

This will create a `pack.sql` file in the `diff-output` directory.

## 6. Apply Migration Pack

The `apply` command applies a migration pack to a database. We'll apply the generated `pack.sql` to the `db_apply` database.

First, you'll need a configuration file to connect to `db_apply`. You can create a new file, e.g., `deepdiffdb.apply.config.yaml`:

```yaml
prod:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3312
  user: "root"
  password: "rootpassword"
  database: "testdb"
```

Then, run the `apply` command:

```bash
deepdiffdb apply -config deepdiffdb.apply.config.yaml -pack diff-output/migration_pack.sql
```

## 7. Verify the Migration

After applying the migration pack, `db_apply` should be identical to `db_target`. You can verify this by running `diff` again with a new configuration file that compares `db_apply` and `db_target`.

Create a new configuration file, e.g., `deepdiffdb.verify.config.yaml`:

```yaml
prod:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3312
  user: "root"
  password: "rootpassword"
  database: "testdb"

dev:
  driver: "mysql"
  host: "127.0.0.1"
  port: 3311
  user: "root"
  password: "rootpassword"
  database: "testdb"
```

Now, run `diff`:

```bash
deepdiffdb diff -config deepdiffdb.verify.config.yaml
```

This command should not report any differences.

---

## Understanding the Three YAML Configuration Files

The `02-advanced-migrations` sample uses three separate YAML files to configure `deepdiffdb` for different stages of the migration workflow. This approach makes the process clearer, more flexible, and reduces the chance of errors.

Here’s the role of each file:

### 1. `deepdiffdb.config.yaml`

-   **Purpose**: **Detecting differences and generating the migration pack.**
-   **Configuration**:
    -   `prod`: Connects to `db_source` (the original database on port 3310).
    -   `dev`: Connects to `db_target` (the desired state on port 3311).
-   **Used by commands**: `check`, `schema-diff`, `diff`, and `gen-pack`.

This is the main configuration file used to compare your starting point (`db_source`) with your desired state (`db_target`).

### 2. `deepdiffdb.apply.config.yaml`

-   **Purpose**: **Applying the migration to the test database.**
-   **Configuration**:
    -   `prod`: Connects to `db_apply` (the clean copy of `db_source` on port 3312).
    -   `dev`: (Dummy entry) While the `apply` command only uses the `prod` database, `deepdiffdb` currently expects a `dev` section to be present in the configuration.
-   **Used by command**: `apply`.

This file tells `deepdiffdb` where to apply the `pack.sql` migration script. Using a separate file explicitly targets the `db_apply` database, ensuring that your original `db_source` is unaffected during testing.

### 3. `deepdiffdb.verify.config.yaml`

-   **Purpose**: **Verifying that the migration was successful.**
-   **Configuration**:
    -   `prod`: Connects to `db_apply` (the migrated database on port 3312).
    -   `dev`: Connects to `db_target` (the desired state on port 3311).
-   **Used by command**: `diff`.

This final configuration is used to compare the database that has just been migrated (`db_apply`) with the database representing your ultimate goal (`db_target`). If the `diff` command reports no differences, it confirms that the migration was executed perfectly.

### Why Use Three Files?

Using separate configuration files for each major step (Difference Detection/Generation, Application, Verification) makes the migration process more robust and easier to manage. Each file serves a single, clear purpose, which helps to prevent errors, especially in complex migration scenarios.