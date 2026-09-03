# Prompt Audit session analysis and bounded chat retention

Prompt Audit currently stores complete content directly on event rows and
stores complete context in a separate table. This repeats long conversation
history for every detection event and makes a database backup carry the largest
payloads.

This change groups retained content below a user-scoped opaque session, dedupes
identical content within that session, adds an administrator analysis action
for the selected event's session, expires unselected Pass content after seven
days while preserving indefinite risk and selected-Pass evidence,
and excludes content tables from logical backups. Detection metadata and the
existing manual cleanup workflow remain available.
