# Design: server-enforced Prompt Audit scanner selection

## Normalization boundary

`ParseQwen3Guard` first classifies returned labels as known or unknown. It then
intersects known categories with the canonical configured scanner set before
constructing the normalized result. Disabled known categories are discarded;
they do not appear in categories, matched scanners, scores, evidence, issue
summaries, logs, or event JSON.

Decision mapping uses only effective categories plus the existing unknown
signal:

| Guard safety | Effective enabled category | Unknown category | Decision |
| --- | --- | --- | --- |
| Safe | any | any | Pass / Allow |
| Controversial | none | none | Pass / Allow |
| Controversial | present | no | Flag / Warn, with existing elevated-category rules |
| Controversial | any | present | Flag / Warn |
| Unsafe | none | none, but Guard returned disabled known categories | Pass / Allow |
| Unsafe | present | any | Critical / Block |
| Unsafe | none | present | Critical / Block |
| Unsafe | no recognized category | no | Critical / Block |

Unknown identifiers remain stable hashes and never expose the returned label.

## Historical reads

Existing rows may contain disabled known categories in `categories` while
`matched_scanners` already records the effective policy intersection. Event
repository reads replace the response-side categories with a copy of matched
scanners before deriving issue summaries. Database rows remain immutable.
