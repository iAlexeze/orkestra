import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

/**
 * Orkestra documentation sidebar.
 *
 * Structure mirrors mkdocs.yml exactly so the two doc systems stay in sync.
 * Doc IDs are the file path relative to the docs/ directory, without extension.
 *
 * To add a new page:
 *  1. Add the .md file to docs/ (existing location)
 *  2. Add the ID here in the correct category
 *  That's all — works for both MkDocs and Docusaurus simultaneously.
 */
const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'doc',
      id: 'index',
      label: 'Overview',
    },

    // ── Basics ──────────────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Basics',
      items: [
        'basics/kubernetes-basics',
        'basics/understanding-orkestra',
      ],
    },

    // ── Getting Started ─────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Getting Started',
      link: {type: 'doc', id: 'getting-started/index'},
      items: [
        'getting-started/writing-your-first-katalog',
        'getting-started/writing-your-first-komposer',
        'getting-started/basic-reconciliation',
      ],
    },

    // ── Runtime Manual ──────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Runtime Manual',
      link: {type: 'doc', id: 'runtime-manual/index'},
      items: [
        // Architecture
        {
          type: 'category',
          label: 'Architecture',
          link: {type: 'doc', id: 'runtime-manual/architecture/index'},
          items: [
            'runtime-manual/architecture/runtime-flow',
            'runtime-manual/architecture/crd-lifecycle',
            'runtime-manual/architecture/design-philosophy',
            'runtime-manual/architecture/architecture-diagrams',
            'runtime-manual/architecture/full-architecture-view',
          ],
        },

        // Concepts
        {
          type: 'category',
          label: 'Concepts',
          link: {type: 'doc', id: 'runtime-manual/concepts/index'},
          items: [
            'runtime-manual/concepts/katalog',
            'runtime-manual/concepts/komposer',
            'runtime-manual/concepts/provider',
            'runtime-manual/concepts/status-management',
            'runtime-manual/concepts/conditional-status-fields',
            'runtime-manual/concepts/conditional-provisioning',
            'runtime-manual/concepts/conditional-webhooks',
            'runtime-manual/concepts/notes',
            'runtime-manual/concepts/validation',
            'runtime-manual/concepts/mutation',
            'runtime-manual/concepts/versioning',

            // Registry Sources
            {
              type: 'category',
              label: 'Registry Sources',
              link: {type: 'doc', id: 'runtime-manual/concepts/registry-sources/index'},
              items: [
                'runtime-manual/concepts/registry-sources/pattern-structure',
                {
                  type: 'category',
                  label: 'Fields',
                  items: [
                    'runtime-manual/concepts/registry-sources/url',
                    'runtime-manual/concepts/registry-sources/version',
                    'runtime-manual/concepts/registry-sources/oci',
                    'runtime-manual/concepts/registry-sources/useKomposer',
                    'runtime-manual/concepts/registry-sources/auth',
                  ],
                },
                'runtime-manual/concepts/registry-sources/multiple-sources',
                'runtime-manual/concepts/registry-sources/best-practices-registry',
                'runtime-manual/concepts/registry-sources/publishing-a-pattern',
                'runtime-manual/concepts/registry-sources/error-reference',
              ],
            },

            'runtime-manual/concepts/hooks',
            'runtime-manual/concepts/constructors',
            'runtime-manual/concepts/observability-model',
            'runtime-manual/concepts/operator-patterns',
            'runtime-manual/concepts/runtime',
            'runtime-manual/concepts/versioning',
            'runtime-manual/concepts/conditional-webhooks',
            'runtime-manual/concepts/conditional-provisioning',
            'runtime-manual/concepts/templating',
            'runtime-manual/concepts/dependency-model',
            'runtime-manual/concepts/health-subsystem',
            'runtime-manual/concepts/reconciler-model',
          ],
        },
      ],
    },

    // ── Guides ──────────────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Guides',
      link: {type: 'doc', id: 'guides/index'},
      items: [
        // User Guide
        {
          type: 'category',
          label: 'User Guide',
          link: {type: 'doc', id: 'guides/user-guide/index'},
          items: [
            'guides/user-guide/how-reconciliation-works',
            'guides/user-guide/multi-version-crd-explained',
            'guides/user-guide/api-conversion',
            'guides/user-guide/extending-orkestra',
            'guides/user-guide/deployment',
            'guides/user-guide/certificate-with-cert-manager',
            'guides/user-guide/certificate-with-openssl',
            'guides/user-guide/choosing-katalog-vs-komposer',
            'guides/user-guide/best-practices',
            'guides/user-guide/writing-hooks',
            'guides/user-guide/testing-operators',
            'guides/user-guide/writing-custom-reconcilers',
          ],
        },

        // Developer Guide
        {
          type: 'category',
          label: 'Developer Guide',
          link: {type: 'doc', id: 'guides/developer-guide/index'},
          items: [
            'guides/developer-guide/development-environment',
            'guides/developer-guide/migrating-from-kubebuilder',

            // Technical Docs
            {
              type: 'category',
              label: 'Technical Docs',
              link: {type: 'doc', id: 'technical-docs/index'},
              items: [
                'technical-docs/generic-reconciler',
                'technical-docs/informer-factory',
                'technical-docs/constructor',
                'technical-docs/konstructor',
                'technical-docs/hooks',
                'technical-docs/katalog',
                'technical-docs/merger',
                'technical-docs/health-server',
                'technical-docs/kordinator',
                'technical-docs/ork-generate',
                'technical-docs/orkestra-registry',
                'technical-docs/conversion-validation-mutation',
                'technical-docs/CONTRIBUTING',
              ],
            },

            // CLI Reference
            {
              type: 'category',
              label: 'CLI Reference',
              link: {type: 'doc', id: 'reference/cli/index'},
              items: [
                {type: 'doc', id: 'reference/cli/init', label: 'ork init'},
                {type: 'doc', id: 'reference/cli/validate', label: 'ork validate'},
                {type: 'doc', id: 'reference/cli/template', label: 'ork template'},
                {type: 'doc', id: 'reference/cli/generate-runtime', label: 'ork generate registry'},
                {type: 'doc', id: 'reference/cli/run', label: 'ork run'},
                {type: 'doc', id: 'reference/cli/status', label: 'ork status'},
                {type: 'doc', id: 'reference/cli/get', label: 'ork get'},
                {type: 'doc', id: 'reference/cli/describe', label: 'ork describe'},
                {type: 'doc', id: 'reference/cli/reconcile', label: 'ork reconcile'},
                {type: 'doc', id: 'reference/cli/events', label: 'ork events'},
                {type: 'doc', id: 'reference/cli/version', label: 'ork version'},
                'reference/cli/inspect-live-crd',
              ],
            },
          ],
        },
      ],
    },

    // ── Reference ────────────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Reference',
      link: {type: 'doc', id: 'reference/index'},
      items: [
        'reference/katalog-schema',
        'reference/katalog-schema-status',
        'reference/komposer-schema',
        'reference/registry-schema',
        'reference/validation-schema',
        'reference/mutation-schema',
        'reference/runtime',
        'reference/metrics',
        'reference/shutdown',
      ],
    },

    // ── Orkestra Registry ────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Orkestra Registry',
      link: {type: 'doc', id: 'orkestra-registry/index'},
      items: [
        'orkestra-registry/how-it-works',
        'orkestra-registry/vision',
      ],
    },

    // ── Orkestra Registry ────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Blog',
      // link: {type: 'doc', id: 'blog/index'},
      items: [
        'blog/your-crd-is-enough',
        'blog/why-orkestra',
        'blog/no-autosync-by-design',
        'blog/validation-is-not-enough.md',
        'blog/security-by-default.md',
        'blog/yaml-as-a-language.md',
      ],
    },

    // ── Publications ─────────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Publications',
      link: {type: 'doc', id: 'publications/index'},
      items: [
        // 'publications/declarative-operators-whitepaper',
        'publications/from-programs-to-data',
        'publications/declarative-conversion',
        'publications/declarative-state-machines',
        'publications/introducing-orkestra-notes',
        'publications/no-trace-runtime',
        'publications/one-runtime-many-crds',
        'publications/operator-sprawl-problem',
        'publications/orkestra-and-gitops',
        'publications/universal-observer-whitepaper',
        'publications/reconcile-contract',
        'publications/when-conditions-paper',
        'publications/value-proposition',
        'publications/orkestra-registry',
        'publications/provider-library',
        'publications/no-autosync-by-design',
        'publications/trust-and-failure-model',
        'publications/metrics-analysis',
      ],
    },

    // ── Use Cases ────────────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'Use Cases',
      link: {type: 'doc', id: 'use-cases/index'},
      items: [
        'use-cases/zero-code-operators',
        'use-cases/platform-namespace-provisioning',
        'use-cases/secret-distribution',
        'use-cases/dependency-ordering',
        'use-cases/centralized-configuration',
        'use-cases/environment-overrides',
        'use-cases/helm-driven-operators',
        'use-cases/multi-team-composition',
        'use-cases/progressive-rollout',
        'use-cases/disaster-recovery',
        'use-cases/air-gapped',
        'use-cases/observability',
        'use-cases/registry',
        'use-cases/conversion',
        'use-cases/hooks',
        'use-cases/custom-constructors',
      ],
    },

    // ── FAQs ─────────────────────────────────────────────────────────────────
    {
      type: 'category',
      label: 'FAQs',
      link: {type: 'doc', id: 'faqs/index'},
      items: ['faqs/why-not-crds'],
    },

    // ── Standalone pages ─────────────────────────────────────────────────────
    {type: 'doc', id: 'security', label: 'Security'},
    {type: 'doc', id: 'support', label: 'Support'},
    {type: 'doc', id: 'roadmap', label: 'Roadmap'},
  ],
};

export default sidebars;
