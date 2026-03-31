/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    'intro',
    {
      type: 'category', label: 'Getting Started', collapsed: false,
      items: ['getting-started/installation', 'getting-started/quick-start', 'getting-started/configuration'],
    },
    {
      type: 'category', label: 'Commands',
      items: ['commands/index', 'commands/check', 'commands/schema-diff', 'commands/schema-migrate', 'commands/diff', 'commands/gen-pack', 'commands/apply', 'commands/resolve-conflicts', 'commands/version'],
    },
    {
      type: 'category', label: 'Database Support',
      items: ['databases/mysql', 'databases/postgresql', 'databases/sqlite', 'databases/mssql', 'databases/oracle'],
    },
    {
      type: 'category', label: 'Features',
      items: ['features/git-versioning', 'features/streaming', 'features/html-reports', 'features/conflict-resolution', 'features/checkpoint-resume', 'features/logging'],
    },
    {
      type: 'category', label: 'Deployment',
      items: ['deployment/docker', 'deployment/cicd', 'deployment/performance', 'deployment/migration-guide'],
    },
    'samples/index',
    'troubleshooting',
    'faq',
  ],
};
module.exports = sidebars;
