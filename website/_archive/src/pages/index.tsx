import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import HomepageFeatures from '@site/src/components/HomepageFeatures';

import styles from './index.module.css';

function HomepageHero(): React.JSX.Element {
  const {siteConfig} = useDocusaurusContext();

  return (
    <header className={clsx('hero', styles.heroBanner)}>
      <div className="container">
        <img
          src="img/logo.png"
          alt="Orkestra"
          className={styles.heroLogo}
        />
        <Heading as="h1" className={styles.heroTitle}>
          {siteConfig.title}
        </Heading>
        <p className={styles.heroSubtitle}>{siteConfig.tagline}</p>
        <p className={styles.heroDescription}>
          Build production-grade Kubernetes operators without writing reconcile loops.
          Declare what resources to create — Orkestra handles the rest.
        </p>
        <div className={styles.buttons}>
          <Link className={styles.btnPrimary} to="/docs/getting-started/">
            Get Started →
          </Link>
          <Link className={styles.btnSecondary} to="/docs/">
            Read the Docs
          </Link>
          <Link
            className={styles.btnSecondary}
            href="https://github.com/orkspace/orkestra">
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

function QuickStart(): React.JSX.Element {
  return (
    <section className={styles.quickStart}>
      <div className="container">
        <Heading as="h2" className={styles.sectionTitle}>
          Get running in minutes
        </Heading>
        <p className={styles.sectionSubtitle}>
          Install the Orkestra CLI, write a Katalog, and your operator is running.
        </p>
        <div className={styles.codeGrid}>
          <div className={styles.codeStep}>
            <div className={styles.stepNumber}>1</div>
            <div className={styles.stepContent}>
              <h3>Install</h3>
              <pre className={styles.codeBlock}>
                <code>{`# Install the ork CLI
curl -sSL https://install.orkestra.sh | bash

# Or via Homebrew
brew install ork`}</code>
              </pre>
            </div>
          </div>
          <div className={styles.codeStep}>
            <div className={styles.stepNumber}>2</div>
            <div className={styles.stepContent}>
              <h3>Write a Katalog</h3>
              <pre className={styles.codeBlock}>
                <code>{`# katalog.yaml
meta:
  name: my-operator
  version: v1

crds:
  - name: website
    apiTypes:
      group: apps.myorg.io
      version: v1
      kind: Website
    operatorBox:
      onCreate:
        deployments:
          - name: "{{ .metadata.name }}"
            image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"`}</code>
              </pre>
            </div>
          </div>
          <div className={styles.codeStep}>
            <div className={styles.stepNumber}>3</div>
            <div className={styles.stepContent}>
              <h3>Run</h3>
              <pre className={styles.codeBlock}>
                <code>{`# Validate your Katalog
ork validate

# Run the operator
ork run`}</code>
              </pre>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

export default function Home(): React.JSX.Element {
  const {siteConfig} = useDocusaurusContext();

  return (
    <Layout
      title={`${siteConfig.title} — ${siteConfig.tagline}`}
      description="Zero-code Kubernetes Operator Runtime. Build production-grade operators by declaring what resources to create — no reconcile loops required.">
      <HomepageHero />
      <main>
        <HomepageFeatures />
        <QuickStart />
      </main>
    </Layout>
  );
}
