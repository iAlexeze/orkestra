# Example Workflows

This section provides a progressive set of example workflows that guide you from simple declarative operators to advanced, production‑ready patterns. Each workflow builds on the previous one, introducing new concepts gradually and showing how they fit together in real implementations.

The goal is to take you from basic familiarity to practical mastery by walking through complete, end‑to‑end examples rather than isolated snippets.

---

## How to Use This Section

!!! tip
    Follow the workflows **in order**.  
    Each example introduces only one or two new ideas, and later examples assume you understand the earlier ones.

The examples are designed as a learning path:

1. Begin with the declarative basics  
2. Build confidence with intermediate workflows  
3. Move into advanced operator patterns  

You can jump around if you already know the fundamentals, but the intended experience is sequential.

---

## What You Will Learn

### Beginner Workflows  
These examples introduce the core ideas behind Orkestra using **purely declarative katalogs and komposers**:

- Creating multi‑resource operators  
- Using templates to generate Kubernetes resources  
- Understanding drift correction  
- Introducing dependencies between CRDs  
- Structuring katalogs for clarity and reuse  

!!! note
    Beginner workflows assume no prior experience with operators beyond what you completed in **[Getting Started](../getting-started/getting-started.md)**.

---

### Intermediate Workflows  
Once you understand the basics, you will build more capable operators:

- Adding multiple resources to a reconcile cycle  
- Using drift correction effectively  
- Adding finalizers  
- Working with typed CRDs  
- Writing simple Go hooks for conditional logic or external data  
- Enriching templates with hook‑generated values  

These workflows reflect common real‑world operator patterns.

---

### Advanced Workflows  
These examples explore the full power of Orkestra:

- Multi‑version CRDs  
- Declarative conversion rules  
- Complex reconciliation logic  
- Custom reconcilers  
- Integrating external systems  
- Building reusable katalogs for teams  
- Structuring registries for production environments  

!!! caution
    Advanced workflows assume you are comfortable with CRD design, templating, hooks, and the Orkestra runtime model.

---

## Requirements

!!! note "Before You Begin"
    These workflows assume you have completed the **Getting Started** section and have already:
    
    - Installed Orkestra  
    - Written your first **Katalog**  
    - Written your first **Komposer**  
    - Run Orkestra locally  
    - Reconciled a simple CR using your katalog  

!!! tip
    Go is **not required** unless you plan to follow typed‑CRD or hook‑based workflows.

---

## Why These Workflows Matter

Orkestra is a declarative operator runtime.  
The best way to understand it is to **see it in action**:

- How a CRD becomes a managed operator  
- How katalogs define behavior  
- How templates generate resources  
- How reconciliation keeps the cluster in sync  
- How dependencies and drift correction work  
- How multi‑version CRDs evolve safely  
- How hooks and reconcilers extend declarative logic  

These workflows show the complete lifecycle, not just the configuration.

---

## Next Steps

Begin with the **Beginner Workflows** to explore how real operators are built using only declarative katalogs and komposers.

These examples expand on what you learned in Getting Started and show how to structure multi‑resource operators, introduce dependencies, and use drift correction — all without writing any Go code.
