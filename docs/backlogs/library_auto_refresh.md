# Backlog: Library Auto & Bulk Refresh

> **Status**: Proposed / Backlog  
> **Target Components**: `internal/library`, `internal/api`, `web/src`

---

## Overview

Automate library metadata and chapter updates across saved titles:
- **Background Worker**: Scheduled background process to periodically check upstream providers for new chapters.
- **Bulk Refresh Action**: Manual "Check for Updates" trigger in the Web UI to refresh all library manga concurrently.
- **Notification & Unread Indicators**: Automatically update unread chapter counters and mark titles with new releases.
