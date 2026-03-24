# Security Policy

## Supported Versions

Orkestra is currently in active development.  
Security fixes will be applied to the latest minor release.

| Version | Supported |
|---------|-----------|
| v0.x.x  | ✔ Active |
| < v0.x  | ✖ Unsupported |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately.

Email: **security@orkestra.io**  
(If this address is not yet active, use GitHub private security advisories.)

Please include:

- A detailed description of the issue  
- Steps to reproduce  
- Potential impact  
- Any suggested fixes  

We will acknowledge your report within **72 hours**.

## Disclosure Policy

- We follow **responsible disclosure** practices.  
- We will work with you to validate and fix the issue.  
- We will publish a security advisory once a fix is available.  
- You will be credited unless you request anonymity.

## What Is Considered a Security Issue?

- Privilege escalation  
- Unauthorized resource access  
- CRD or cluster‑wide compromise  
- Remote code execution  
- Bypass of Orkestra’s isolation guarantees  

## What Is *Not* Considered a Security Issue?

- Misconfigured RBAC in user clusters  
- Incorrect Katalog definitions  
- Expected Kubernetes behavior  
- Resource exhaustion caused by user‑defined CRDs  
