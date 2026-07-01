# orkspace/charts

A centralized repository for all **shared Helm charts** used across the orkspace ecosystem.  
This repository provides a single, versioned source of truth for platform‑level charts that can be consumed by applications, operators, and internal tooling.

The goal of this repository is to make Helm‑based components easy to publish, discover, and reuse across orkspace projects.

---

## Purpose

This repository exists to:

- Host reusable Helm charts maintained by the orkspace platform team.
- Provide a consistent publishing workflow for Git‑based and OCI‑based charts.
- Serve as the upstream source for charts referenced by Orkestra Komposer (`sources.helm`).
- Enable teams to version, distribute, and consume charts without maintaining separate repositories.

---

## What Belongs Here

- Shared infrastructure charts  
- Platform‑level building blocks  
- Example or reference charts  
- Charts intended for consumption by Orkestra patterns  
- Charts published to Artifact Hub

This repository is **not** intended for application‑specific charts or one‑off deployments.

---

## Contributing

- Follow semantic versioning for chart updates.
- Include a README in each chart directory.
- Ensure charts render cleanly with `helm template`.
- Validate charts before publishing.