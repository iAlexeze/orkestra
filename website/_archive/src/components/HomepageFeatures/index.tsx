import React from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  icon: string;
  description: React.JSX.Element;
};

const features: FeatureItem[] = [
  {
    title: 'Zero-code operators',
    icon: '⚡',
    description: (
      <>
        Write a Katalog YAML — Orkestra generates and runs the full operator. No
        reconcile loops, no client-go boilerplate, no controller-runtime wiring.
      </>
    ),
  },
  {
    title: 'One runtime, many CRDs',
    icon: '🔀',
    description: (
      <>
        Register 'N' number of CRDs in a single operator binary. Each CRD gets its
        own informers, reconcile workers, queue, health state, metrics, and events — automatically.
      </>
    ),
  },
  // {
  //   title: 'Declarative validation',
  //   icon: '✅',
  //   description: (
  //     <>
  //       Define field constraints in YAML. Orkestra enforces them at admission time
  //       (webhook) and at reconcile time — the same rules, declared once.
  //     </>
  //   ),
  // },
  // {
  //   title: 'Declarative mutation',
  //   icon: '🔧',
  //   description: (
  //     <>
  //       Set defaults and overrides in the Katalog. Applied at admission time and
  //       corrected on every reconcile loop without writing a single line of Go.
  //     </>
  //   ),
  // },
  {
    title: 'Multi-version CRDs',
    icon: '🔄',
    description: (
      <>
        Declare CRD versions in the Katalog. Orkestra handles version
        negotiation between v1alpha1, v1beta1, and v1 with a simple mapping spec.
      </>
    ),
  },
  // {
  //   title: 'Dependency ordering',
  //   icon: '📦',
  //   description: (
  //     <>
  //       CRDs declare <code>dependsOn</code>. Orkestra resolves the graph and starts
  //       workers in topological order — cycle detection included.
  //     </>
  //   ),
  // },
  // {
  //   title: 'Built-in health API',
  //   icon: '🩺',
  //   description: (
  //     <>
  //       Every operator exposes <code>/health</code> and <code>/katalog/{'{crd}'}</code>
  //       {' '}endpoints out of the box. Monitor operator and per-CRD health without extra
  //       code.
  //     </>
  //   ),
  // },
  // {
  //   title: 'Go hooks when you need them',
  //   icon: '🪝',
  //   description: (
  //     <>
  //       Need custom logic? Register typed Go hooks alongside your Katalog. Orkestra
  //       calls them at the right lifecycle point — no framework to fight.
  //     </>
  //   ),
  // },
  // {
  //   title: 'GitOps & Komposer',
  //   icon: '🗂️',
  //   description: (
  //     <>
  //       Compose multiple Katalog files with Komposer. Pull patterns from Git or OCI
  //       registries. Everything is declarative and version-pinned.
  //     </>
  //   ),
  // },
];

function Feature({title, icon, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4', styles.featureCol)}>
      <div className={styles.featureCard}>
        <div className={styles.featureIcon}>{icon}</div>
        <Heading as="h3" className={styles.featureTitle}>
          {title}
        </Heading>
        <p className={styles.featureDescription}>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): React.JSX.Element {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className={styles.featuresHeader}>
          <Heading as="h2">Why Orkestra?</Heading>
          <p>
            The Kubernetes operator pattern is powerful — but implementing it correctly
            takes weeks. Orkestra reduces it to minutes with your <strong>CRD</strong>.
            {/* Orkestra handles the framework so you focus on your domain. */}
          </p>
        </div>
        <div className="row">
          {features.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
