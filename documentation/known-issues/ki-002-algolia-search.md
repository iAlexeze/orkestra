# KI-002 — Algolia site search returns no results

**Status:** Open — pre-v1  
**Affects:** orkestra.sh — the search bar on all documentation pages

---

## What you see

Typing in the search box on orkestra.sh produces no results. The input is accepted but nothing comes back.

---

## Why it happens

Orkestra's documentation search is powered by Algolia DocSearch. The DocSearch crawler must index the site before results appear. For sites in early access, the index is either not yet populated or the crawler has not completed a full pass.

This is a configuration and timing issue, not a problem with the documentation content itself.

---

## Workaround

Use your browser's built-in search (`Ctrl+F` / `Cmd+F`) on a page, or navigate directly via the sidebar. The full documentation is available — it is only the cross-page search that is non-functional.

---

## Resolution plan

Verify the Algolia DocSearch index is configured with the correct site URL and that the crawler has completed at least one successful run. Once the index is populated, search will work without any changes to the documentation source.

Scheduled for resolution before v1.
