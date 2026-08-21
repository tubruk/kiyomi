# Provider Binding

A manga in kiyomi may be enriched by zero or more external providers. Of
these, at most one is designated the **content provider** — the source
whose chapter list and pages are downloaded. Others may serve only
metadata (aliases, cover, description, tags, authors).

Three operations govern the relationship: **add** a provider binding,
**switch** which provider is the content one, and **remove** a binding.

This document captures the design decisions behind these operations. The
intent is **why** they behave the way they do, not how they are wired
in code.

---

## The model

A manga carries a set of provider bindings. Each binding points at one
provider and one remote series. Exactly one binding — at most — may be
flagged as the content source; the rest contribute metadata only.

A provider is "content-capable" if it advertises a `Content` capability
in its plugin contract. A content-capable binding is **eligible** to be
the content source; a non-content-capable binding can only contribute
metadata.

The three operations:

| Operation | Effect |
| --- | --- |
| **Add** | Attach a new binding. May optionally flag it as the content source. |
| **Switch** | Move the content-source flag from one binding to another. |
| **Remove** | Detach a binding. Forbidden for the current content source until another takes its place. |

---

## Decisions

### 1. Preconditions run before state changes

**Decision:** all eligibility checks (capabilities, schema, identity)
must succeed **before** any binding is written.

**Why:** the system has no compensating-delete path. Once a binding is
persisted, a later failure leaves the manga pointing at a provider that
cannot actually serve it. Pre-checking is the only way to guarantee
that any binding on disk is by construction valid.

**Consequence:** new preconditions must be inserted above the write
step. Reviewers should treat any reordering that moves a check below a
state change as a bug.

### 2. Switch is destructive; the chapters directory is the source of truth

**Decision:** switching content providers wipes the existing chapters
and triggers a fresh fetch from the new provider. A switch that
completes with no chapters is acceptable; a switch that fails silently
is not.

**Why:** keeping the previous provider's chapters under a new
content-source flag would be self-contradictory — the chapter list
would say one thing, the binding another. The design accepts "lose all
chapters" as the cost of switching, because the alternative (fetch
first, swap the flag only on success) doubles the work in the common
case and still requires cleanup on failure.

**Consequence:** the response to a switch must distinguish three
outcomes:

- **Success with new chapters** — the new content source is in place
  and the chapter list reflects it.
- **Success with no new chapters** — the new content source is in
  place but produced an empty list (legitimate for some series).
- **Failure** — the chapters directory has been wiped and the operation
  did not complete. The response must signal this clearly so the user
  is not led to believe the manga is intact.

### 3. Author/artist lists are deduped across the singular and plural fields

**Decision:** when a provider returns both a single-name field and a
list field, the merged result is the union with duplicates removed. No
field wins precedence.

**Why:** providers are inconsistent about which field they populate.
Some return only one, some return both with overlapping values, some
return one per shape and the other empty. Treating either field as
authoritative produces duplicates whenever both are present.

**Consequence:** anywhere author or artist lists are assembled from
provider data, the same merge-and-dedupe rule must apply. The import
preview, the conflict view, and the persisted record must all show the
same list for the same input.

### 4. Provider-removal errors are a stability risk

**Decision:** the remove operation distinguishes three failure modes:
"this is the last content-capable binding" (the user must add a
replacement first), "this binding does not exist", and "everything
else".

**Why:** the first two are user-recoverable and the UI needs to react
to them differently from a generic server error. Without explicit
distinction, the UI either disables the action preemptively (false
negatives) or shows a generic error and forces the user to guess
(recovery friction).

**Consequence (transitional):** the current implementation dispatches
on error-message text, which silently breaks if the underlying
messages are reworded. The correct long-term answer is typed error
sentinels in the library package. Until that lands, any change to
library error wording is a behavior change for the remove endpoint.

### 5. User-corrected titles are not overwritten on switch

**Decision:** if the target binding already exists, its stored title
is left as-is. A switch that re-targets an existing binding does not
refresh the title from whatever the provider returned this time.

**Why:** the user may have corrected the title once already. The
provider's current view of the title is not necessarily an improvement
over the user's correction. The default — preserve — is the safer
choice.

**Consequence:** callers that genuinely need a title refresh must
remove the binding and re-add it, or use a separate title-edit
operation. This is a known limitation of the current flow.

### 6. Two proposed capabilities remain unimplemented

Two capabilities have been proposed in the design notes but are not
yet part of the live system:

- A `has_stable_chapter_id` flag on content providers, used to pick a
  refresh-correlation strategy.
- A `number_mapping` extension point on chapter records, used when
  heuristic normalization is insufficient.

**Decision:** both stay in the design space but are not active. They
become relevant only when a concrete use case (provider stability
variation, LLM-assisted normalization, manual overrides) lands.

**Consequence:** any design doc that still describes them as live
behavior is stale. The next doc pass should mark them as
"proposed, not implemented" or move them to a deferred appendix.

---

## Failure-mode summary

| Operation | Precondition failure | Mid-operation failure | Result |
| --- | --- | --- | --- |
| Add | reject, no state change | n/a (write is atomic) | no orphan binding |
| Switch | reject, no state change | report failure to caller, chapter list is empty | user can re-trigger refresh |
| Remove | reject with reason (last content-capable, not found) | n/a | no partial detachment |

---

## Open questions

- Should switch become atomic — i.e., should the chapters directory be
  snapshotted before the wipe and restored on refresh failure?
- What is the user-facing recovery flow for a switch that wiped
  chapters and then failed? Manual re-trigger, or automatic retry?
- Is the user-corrected-title preservation in Decision 5 the right
  default, or should the provider view always win?
- Are the two deferred capabilities in Decision 6 still wanted, or
  should they be retired to keep the design space clean?
