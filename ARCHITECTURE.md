# Architecture

This document records project-level contracts that affect command design,
rendering, and compatibility. Keep it focused on durable rules rather than
implementation notes that belong next to code.

## Output Contract

Golden rule: JSON preserves upstream API semantics. Text and table output may
interpret Slack data for people and agents.

Slack messages can carry readable content in several surfaces: top-level
`text`, Block Kit `blocks`, legacy `attachments`, and `files`. Text and table
renderers may flatten those surfaces, remove duplicated fallback text, resolve
mentions, and truncate long values to keep output useful in a terminal. That
rendered body is CLI-derived content.

JSON output must not overwrite upstream fields with CLI-derived rendering. If
Slack returns `text: ""` for a block-only message, JSON must preserve that empty
`text` value and include modeled readable surfaces such as `blocks`,
`attachments`, and `files` when available. Scripts can then distinguish Slack's
transport data from the CLI's text rendering.

Renderer-derived fields do not belong in default JSON envelopes. If a command
needs a structured diagnostic or control-plane response, it should use a local
carve-out such as `slck config show --json`, not change resource command output
semantics.

Practical consequences:

- `--output text` and table output may show rendered block, attachment, or file
  content.
- JSON must keep included upstream fields at their upstream meaning.
- Missing modeled readable surfaces in JSON is a fidelity bug.
- Adding a rendered `text`, `body`, or similar field to default JSON is a
  compatibility break unless it is explicitly versioned or scoped to a local
  diagnostic command.

Related: #143, #144, #145, #173.
