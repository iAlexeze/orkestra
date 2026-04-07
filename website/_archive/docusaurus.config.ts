import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Orkestra',
  tagline: 'Zero-code Kubernetes Operator Runtime',
  favicon: 'img/logo.png',

  url: 'https://orkestra.sh',
  baseUrl: '/',

  organizationName: 'iAlexeze',
  projectName: 'orkestra',

  // Broken links in the existing docs should warn, not fail the build
  onBrokenLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },
// siteConfig.markdown.hooks.onBrokenMarkdownLinks
  // 'detect' treats .md files as plain Markdown (not MDX) and .mdx as MDX.
  // This prevents MDX from choking on < <= >= and {expr} in existing docs
  // that were written for MkDocs, not React/JSX.
  markdown: {
    format: 'detect',
    mermaid: true,
    hooks:
      {
        onBrokenMarkdownLinks: 'warn',
      }

  },

  // Preload fonts for better performance
  headTags: [
    // Preload Inter variable fonts
    {
      tagName: 'link',
      attributes: {
        rel: 'preload',
        href: '/static/fonts/Inter-Italic-VariableFont-opsz-wght.ttf',
        as: 'font',
        type: 'font/ttf',
        crossorigin: 'anonymous',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'preload',
        href: '/static/fonts/Inter-Italic-VariableFont-opsz-wght.ttf',
        as: 'font',
        type: 'font/ttf',
        crossorigin: 'anonymous',
      },
    },
    // Preload static fallback weights for broader compatibility
    {
      tagName: 'link',
      attributes: {
        rel: 'preload',
        href: '/static/fonts/static/Inter_18pt-Regular.ttf',
        as: 'font',
        type: 'font/ttf',
        crossorigin: 'anonymous',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'preload',
        href: '/static/fonts/static/Inter_18pt-Medium.ttf',
        as: 'font',
        type: 'font/ttf',
        crossorigin: 'anonymous',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'preload',
        href: '/static/fonts/static/Inter_18pt-SemiBold.ttf',
        as: 'font',
        type: 'font/ttf',
        crossorigin: 'anonymous',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'preload',
        href: '/static/fonts/static/Inter_18pt-Bold.ttf',
        as: 'font',
        type: 'font/ttf',
        crossorigin: 'anonymous',
      },
    },
  ],

  themes: [
    '@docusaurus/theme-mermaid',
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
    image: 'img/logo.png',

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
        src: 'img/logo.png',
        srcDark: 'img/logo.png',
      },
      hideOnScroll: false,
      items: [
        {
          to: '/docs/blog/why-orkestra',
          label: 'Why Orkestra?',
          position: 'left',
        },
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          to: '/docs/publications/',
          label: 'Publications',
          position: 'left',
        },
        {
          to: '/docs/blog/',
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

    // Algolia DocSearch Configuration
    //  https://docsearch.algolia.com/apply/
    algolia: {
      appId: '1YJ1C111LH',
      apiKey: 'b4bbac9a499ecc51109f23abd327cf29',      
      indexName: 'orkestra',
      
      // Optional: contextual search adds context to search results
      contextualSearch: true,
      
      // Optional: search parameters
      searchParameters: {},
      
      // Optional: placeholder for search input
      placeholder: 'Search documentation...',
      
      // Optional: path for search page
      searchPagePath: 'search',
      
      // Optional: whether to show search results immediately when typing
      // (default: false)
      // immediateSearch: false,
      
      // Optional: insights plugin for analytics
      // insights: false,
    },

    footer: {
      style: 'dark',
      logo: {
        alt: 'Orkestra',
        src: 'img/logo.png',
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
            {label: 'Blog', href: '/docs/blog'},
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