import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Orkestra',
  tagline: 'Zero-code Kubernetes Operator Runtime',
  favicon: 'img/fav.png',

  url: 'https://orkestra.sh',
  baseUrl: '/',

  organizationName: 'iAlexeze',
  projectName: 'orkestra',

  // Broken links in the existing docs should warn, not fail the build
  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  // 'detect' treats .md files as plain Markdown (not MDX) and .mdx as MDX.
  // This prevents MDX from choking on < <= >= and {expr} in existing docs
  // that were written for MkDocs, not React/JSX.
  markdown: {
    format: 'detect',
    mermaid: true,
  },
  themes: [
    '@docusaurus/theme-mermaid',
    [
      '@easyops-cn/docusaurus-search-local',
      {
        hashed: true,
        language: ['en'],
        highlightSearchTermsOnTargetPage: true,
        explicitSearchResultPath: true,
        docsRouteBasePath: '/docs',
        searchBarPosition: 'left',
        searchBarShortcutHint: true,
      },
    ],
  ],
  presets: [
    [
      'classic',
      {
        docs: {
          // Reuse the existing MkDocs content directory — no duplication needed
          path: '../docs',
          routeBasePath: 'docs',
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/iAlexeze/orkestra/edit/main/',
          showLastUpdateTime: true,
          showLastUpdateAuthor: true,
          // Don't auto-generate sidebar — use explicit sidebars.ts
          sidebarCollapsed: true,
          sidebarCollapsible: true,
          // Exclude non-doc files
          exclude: [
            '**/*.sh',
            '**/*.yaml',
            '**/*.yml',
            '**/scan/**',
            '**/deprecated/**',
            '**/new/**',
          ],
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
        gtag: {
          trackingID: 'G-5Z1VTPDL73',
          anonymizeIP: true,
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/fav.png',

    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: false,
    },

    // Announcement bar — remove once out of alpha
    announcementBar: {
      id: 'alpha',
      content:
        'Orkestra is in active development. <a target="_blank" href="https://github.com/iAlexeze/orkestra/releases">View releases →</a>',
      backgroundColor: '#0e0a14',
      textColor: '#ffffff',
      isCloseable: true,
    },

    navbar: {
      title: 'Orkestra',
      logo: {
        alt: 'Orkestra Logo',
        src: 'img/fav.png',
        srcDark: 'img/fav.png',
      },
      hideOnScroll: false,
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          to: '/docs/reference/',
          label: 'Reference',
          position: 'left',
        },
        {
          to: '/docs/use-cases/',
          label: 'Use Cases',
          position: 'left',
        },
        {
          to: '/docs/publications/',
          label: 'Blog',
          position: 'left',
        },
        {
          href: 'https://github.com/iAlexeze/orkestra',
          position: 'right',
          className: 'header-github-link',
          'aria-label': 'GitHub repository',
        },
      ],
    },

    footer: {
      style: 'dark',
      logo: {
        alt: 'Orkestra',
        src: 'img/fav.png',
        href: 'https://orkestra.sh',
        width: 120,
      },
      links: [
        {
          title: 'Learn',
          items: [
            {label: 'Getting Started', to: '/docs/getting-started/'},
            {label: 'Runtime Manual', to: '/docs/runtime-manual/'},
            {label: 'Use Cases', to: '/docs/use-cases/'},
            {label: 'Publications', to: '/docs/publications/'},
          ],
        },
        {
          title: 'Reference',
          items: [
            {label: 'Katalog Schema', to: '/docs/reference/katalog-schema'},
            {label: 'CLI Reference', to: '/docs/reference/cli/'},
            {label: 'Metrics', to: '/docs/reference/metrics'},
            {label: 'Validation Schema', to: '/docs/reference/validation-schema'},
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/iAlexeze/orkestra',
            },
            {
              label: 'Twitter',
              href: 'https://twitter.com/orkestra',
            },
            {
              label: 'Releases',
              href: 'https://github.com/iAlexeze/orkestra/releases',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {label: 'Blog', href: 'https://blog.orkestra.sh/'},
            {label: 'Security', to: '/docs/security'},
            {label: 'Roadmap', to: '/docs/roadmap'},
            {label: 'Support', to: '/docs/support'},
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Orkestra. Built with Docusaurus.`,
    },

    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'yaml', 'go', 'json', 'docker', 'shell-session'],
    },

    docs: {
      sidebar: {
        hideable: true,
        autoCollapseCategories: true,
      },
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
