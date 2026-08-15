# Import Metadata (Side-by-Side Merge)

> **Status**: Feature spec for external metadata sync. In Phase 1, metadata is managed directly via local edit forms (`EditMetadataDialog`); side-by-side external provider merge endpoints are deferred.

---

## User Journey

When a user has a manga in their library but notices its description is incomplete, tags are missing, or cover art is low quality, they can trigger "Import Metadata" to fetch rich, curated profiles.

### Step 1: Provider Search
1. User clicks **"Fetch Metadata"** from the manga detail page.
2. An overlay opens (a wide panel/drawer or dialog).
3. User selects a metadata provider (AniList, Kitsu, etc.) and searches by title.
4. Kiyomi displays a list of candidate results. The user selects the correct matching series.

### Step 2: Side-by-Side Comparison & Merge (Refined UX)

Once a candidate is selected, Kiyomi fetches the candidate's full profile and displays it alongside the local data.

```
+--------------------------------------------------------------------------------+
| Import Metadata: Solo Leveling                                                 |
+--------------------------------------------------------------------------------+
| Show identical fields [ ]                                                      |
|                                                                                |
| [Field]          [Local Value]                 [Remote Value]        [Action]  |
|                                                                                |
| Title            Solo Leveling                 Only Level Up        (Select L) |
|                                                                                |
| Synopsis         A weak hunter gets...         In a world where...  (Select R) |
|                                                                                |
| Tags             [Action] [Fantasy]            [Action] [Adventure] (Merge)    |
|                  [System]                      [Webtoon]                       |
|                                                                                |
+--------------------------------------------------------------------------------+
|                                                      [ Import Selected ]       |
+--------------------------------------------------------------------------------+
```

#### UX & Layout Constraints:
1. **Wide Panel**: The comparison layout must be wide enough (e.g., `sm:max-w-4xl` or `w-11/12` screen width) to comfortably display long text (like synopsis) side-by-side without truncation or excessive vertical wrapping.
2. **Diff & Collapsing identical fields**:
   - By default, **only fields with differences** are shown.
   - Any identical/unchanged fields are **collapsed** under an expandable summary header at the bottom (e.g. `^ 3 Unchanged Fields`).
   - A toggle checkbox or switch labeled `"Show unchanged fields"` allows the user to expand and inspect identical fields if needed.
3. **Cover Image Exception**:
   - The **Cover Asset** field is **always shown** in the active diff list to allow users to visually compare image quality, even if URLs match.
4. **Visual Selection Primitives & Preselection**:
   - Instead of nested dropdown selects, the user can click directly on the **Local Card** or **Remote Card** to select which value to keep.
   - **Default Selection**: By default, the right-hand card (new remote metadata) is **always preselected** for all fields, **unless the new remote metadata is empty/null**. In that case, the left-hand card (local metadata) is preselected to protect existing local data. The user can manually toggle the selection by clicking the opposite card.
   - The selected card gets a distinct highlighted border (e.g., `border-primary`).
5. **Tag Merge Action & Normalization**:
   - For tags/genres and authors, the UI offers a third action option: **"Merge Both"**. This combines unique elements from both sets.
   - **Tag Normalization**: Normalization (lowercase, trim, prefix stripping) is handled directly in the provider plugin code to ensure the core merge mechanism operates on clean datasets.
6. **Deduplicated Appends**:
   - External links use an append-and-deduplicate strategy to preserve existing trackers while adding new ones.
