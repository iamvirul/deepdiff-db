import React from 'react';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import styles from './index.module.css';

function IconSearch() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  );
}

function IconPackage() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M21 16V8a2 2 0 0 0-1-1.73L13 2.27a2 2 0 0 0-2 0L4 6.27A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
      <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
      <line x1="12" y1="22.08" x2="12" y2="12" />
    </svg>
  );
}

function IconBolt() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
    </svg>
  );
}

function IconShield() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <polyline points="9 12 11 14 15 10" />
    </svg>
  );
}

function IconBarChart() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <line x1="18" y1="20" x2="18" y2="10" />
      <line x1="12" y1="20" x2="12" y2="4" />
      <line x1="6" y1="20" x2="6" y2="14" />
      <line x1="2" y1="20" x2="22" y2="20" />
    </svg>
  );
}

function IconDatabase() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <ellipse cx="12" cy="5" rx="9" ry="3" />
      <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3" />
      <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5" />
    </svg>
  );
}

const features = [
  {
    Icon: IconSearch,
    title: 'Schema Drift Detection',
    description: 'Instantly detect added/removed tables, column type changes, nullability drift, index and foreign key changes between any two databases.',
  },
  {
    Icon: IconPackage,
    title: 'SQL Migration Packs',
    description: 'Auto-generate a fully transactional SQL migration pack. DELETE + INSERT for changed rows, ALTER TABLE for new columns — ready to review and apply.',
  },
  {
    Icon: IconBolt,
    title: 'Streaming Large Datasets',
    description: 'Keyset-paginated batch hashing keeps memory flat at O(batch_size) regardless of table size. Parallel table processing cuts wall-clock time by 4×.',
  },
  {
    Icon: IconShield,
    title: 'Conflict Resolution',
    description: 'Detect rows that differ in both databases. Resolve with configurable ours/theirs/manual strategies or interactively via the CLI.',
  },
  {
    Icon: IconBarChart,
    title: 'Interactive HTML Reports',
    description: 'Generate a self-contained HTML report with schema diff viewer, data diff visualizer, conflict highlights, and SQL migration preview.',
  },
  {
    Icon: IconDatabase,
    title: '5 Database Engines',
    description: 'MySQL, PostgreSQL, SQLite, Microsoft SQL Server, and Oracle Database — all with a single static binary and zero native dependencies.',
  },
];

const databases = [
  { name: 'MySQL', driver: 'mysql', badge: '#e67e22', since: 'v0.1' },
  { name: 'PostgreSQL', driver: 'postgres', badge: '#336791', since: 'v0.1' },
  { name: 'SQLite', driver: 'sqlite', badge: '#0f80cc', since: 'v0.1' },
  { name: 'SQL Server', driver: 'mssql', badge: '#cc2927', since: 'v0.8' },
  { name: 'Oracle', driver: 'oracle', badge: '#f80000', since: 'v0.9' },
];

function FeatureCard({ Icon, title, description }) {
  return (
    <div className={styles.featureCard}>
      <div className={styles.featureIcon}><Icon /></div>
      <h3 className={styles.featureTitle}>{title}</h3>
      <p className={styles.featureDesc}>{description}</p>
    </div>
  );
}

function DatabaseBadge({ name, driver, badge, since }) {
  return (
    <div className={styles.dbBadge}>
      <span className={styles.dbDot} style={{ background: badge }} />
      <span className={styles.dbName}>{name}</span>
      <code className={styles.dbDriver}>{driver}</code>
      <span className={styles.dbSince}>{since}</span>
    </div>
  );
}

export default function Home() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout title="DeepDiff DB" description="Compare databases. Detect drift. Generate safe migrations.">
      {/* Hero */}
      <div className={styles.hero}>
        <div className={styles.heroBg} />
        <div className={styles.heroContent}>
          <div className={styles.heroBadge}>v0.9 — Oracle support now available</div>
          <h1 className={styles.heroTitle}>
            Database diff that<br />
            <span className={styles.heroGradient}>actually ships</span>
          </h1>
          <p className={styles.heroSubtitle}>
            Compare any two databases, detect schema drift and data divergence,
            and generate a safe, reviewable SQL migration pack — in seconds.
          </p>
          <div className={styles.heroCta}>
            <Link className={styles.ctaPrimary} to="/docs/getting-started/installation">
              Get Started →
            </Link>
            <Link className={styles.ctaSecondary} to="/docs/getting-started/quick-start">
              Quick Start
            </Link>
          </div>
          <div className={styles.heroInstall}>
            <code>brew install deepdiff-db</code>
          </div>
        </div>
      </div>

      {/* Database support strip */}
      <div className={styles.dbStrip}>
        <span className={styles.dbStripLabel}>Supports</span>
        {databases.map((db) => (
          <DatabaseBadge key={db.driver} {...db} />
        ))}
      </div>

      {/* Features grid */}
      <div className={styles.section}>
        <h2 className={styles.sectionTitle}>Everything you need to sync databases safely</h2>
        <p className={styles.sectionSub}>
          A single static binary. Zero native dependencies. Works with your existing CI pipeline.
        </p>
        <div className={styles.featuresGrid}>
          {features.map((f) => (
            <FeatureCard key={f.title} {...f} />
          ))}
        </div>
      </div>

      {/* Code example -->*/}
      <div className={styles.codeSection}>
        <div className={styles.codeSectionInner}>
          <div className={styles.codeLeft}>
            <h2>From zero to diff in 60 seconds</h2>
            <p>One config file. Three commands. Your databases are in sync.</p>
            <Link className={styles.ctaPrimary} to="/docs/getting-started/quick-start">
              See the full workflow →
            </Link>
          </div>
          <div className={styles.codeRight}>
            <div className={styles.codeBlock}>
              <div className={styles.codeHeader}>
                <span className={styles.dot} style={{background:'#ff5f57'}}/>
                <span className={styles.dot} style={{background:'#febc2e'}}/>
                <span className={styles.dot} style={{background:'#28c840'}}/>
                <span className={styles.codeLang}>bash</span>
              </div>
              <pre className={styles.codePre}>{`# 1. Check connections
deepdiffdb check

# 2. Run a full diff
deepdiffdb diff --html

# 3. Generate migration pack
deepdiffdb gen-pack

# 4. Review + apply
deepdiffdb apply \\
  --pack diff-output/migration_pack.sql`}</pre>
            </div>
          </div>
        </div>
      </div>

      {/* Footer CTA -->*/}
      <div className={styles.footerCta}>
        <h2>Ready to bring your databases in sync?</h2>
        <div className={styles.heroCta}>
          <Link className={styles.ctaPrimary} to="/docs/getting-started/installation">
            Install DeepDiff DB
          </Link>
          <a className={styles.ctaSecondary} href="https://github.com/iamvirul/deepdiff-db">
            View on GitHub
          </a>
        </div>
      </div>
    </Layout>
  );
}
