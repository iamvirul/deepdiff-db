// @ts-check
const { themes: prismThemes } = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'DeepDiff DB',
  tagline: 'Compare databases. Detect drift. Generate safe migrations.',
  favicon: 'img/logo.svg',
  url: 'https://iamvirul.github.io',
  baseUrl: '/deepdiff-db/',
  organizationName: 'iamvirul',
  projectName: 'deepdiff-db',
  trailingSlash: false,
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  i18n: { defaultLocale: 'en', locales: ['en'] },
  presets: [['classic', {
    docs: {
      sidebarPath: require.resolve('./sidebars.js'),
      editUrl: 'https://github.com/iamvirul/deepdiff-db/edit/main/website/',
      showLastUpdateTime: true,
    },
    blog: {
      showReadingTime: true,
      blogTitle: 'DeepDiff DB Blog',
      blogDescription: 'Behind the scenes of building DeepDiff DB',
      postsPerPage: 10,
      editUrl: 'https://github.com/iamvirul/deepdiff-db/edit/main/website/',
    },
    theme: { customCss: require.resolve('./src/css/custom.css') },
  }]],
  themeConfig: {
    navbar: {
      title: 'DeepDiff DB',
      logo: { alt: 'DeepDiff DB', src: 'img/logo.svg' },
      items: [
        { type: 'docSidebar', sidebarId: 'docs', position: 'left', label: 'Docs' },
        { to: '/blog', label: 'Blog', position: 'left' },
        { href: 'https://github.com/iamvirul/deepdiff-db', label: 'GitHub', position: 'right' },
        { href: 'https://github.com/iamvirul/deepdiff-db/releases', label: 'v1.3', position: 'right' },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        { title: 'Docs', items: [
          { label: 'Installation', to: '/docs/getting-started/installation' },
          { label: 'Quick Start', to: '/docs/getting-started/quick-start' },
          { label: 'Configuration', to: '/docs/getting-started/configuration' },
        ]},
        { title: 'Reference', items: [
          { label: 'Commands', to: '/docs/commands/' },
          { label: 'Database Support', to: '/docs/databases/mysql' },
          { label: 'Changelog', href: 'https://github.com/iamvirul/deepdiff-db/blob/main/CHANGELOG.md' },
        ]},
        { title: 'Project', items: [
          { label: 'GitHub', href: 'https://github.com/iamvirul/deepdiff-db' },
          { label: 'Issues', href: 'https://github.com/iamvirul/deepdiff-db/issues' },
          { label: 'Roadmap', href: 'https://github.com/iamvirul/deepdiff-db/blob/main/ROADMAP.md' },
        ]},
      ],
      copyright: `Copyright © ${new Date().getFullYear()} iamvirul. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'yaml', 'sql'],
    },
    colorMode: { defaultMode: 'light', disableSwitch: false, respectPrefersColorScheme: true },
  },
};
module.exports = config;
