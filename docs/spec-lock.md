# Spec Lock — MCP Tasks Extension (io.modelcontextprotocol/tasks)

Locked against: https://tasks.extensions.modelcontextprotocol.io/
(SEP-2663, part of the 2026-07-28 MCP specification release)

## Extension identifier
`io.modelcontextprotocol/tasks`

## Supported methods (exactly 3 — no more, no less)
- `tasks/get`    — poll for status/result. Idempotent, cacheable.
- `tasks/update` — client fulfills inputRequests. Returns empty ack.
- `tasks/cancel` — client signals cancel intent. Returns empty ack. Cooperative only.

There is intentionally NO `tasks/list`. Do not add one to the wire protocol.

## Task-augmentable request types (currently)
- `tools/call` only.

## State machine

    working ────────► completed   (terminal)
       │  ▲       ╲
       │  │        ╲──► failed     (terminal)
       ▼  │         ╲
  input_required     ╲─► cancelled (terminal)
       │  (tasks/update)
       └──────────────► working

Legal transitions:
  working         -> input_required   (server needs client input)
  working         -> completed        (success incl. isError:true results)
  working         -> failed           (JSON-RPC execution error)
  working         -> cancelled        (cancel took effect)
  input_required  -> working          (tasks/update received, still processing)
  input_required  -> completed/failed/cancelled (same terminal rules apply)

Illegal transitions (must error, never silently allow):
  completed  -> anything
  failed     -> anything
  cancelled  -> anything
  (all terminal states are final — no transitions out)

## Key invariants
1. CreateTaskResult MUST NOT be returned until the task is durably
   findable via tasks/get. No "create then eventually consistent."
2. Task IDs MUST have sufficient entropy to resist enumeration —
   they double as bearer tokens. No separate auth token needed.
3. failed = JSON-RPC protocol error during execution.
   completed (with isError:true) = tool-level error. These are NOT
   interchangeable — a tool returning isError:true is still "completed."
4. inputRequests keys are unique for the lifetime of one task; never
   reused even after being answered.
5. tasks/update ack is eventually consistent — status may not reflect
   the update immediately after the ack returns.
6. tasks/cancel is cooperative — server may ignore it, task may reach
   a non-cancelled terminal state anyway.
7. Streamable HTTP transport: tasks/get, tasks/update, tasks/cancel
   MUST set the Mcp-Name header to params.taskId, and Mcp-Method to
   the JSON-RPC method name. (Wire this in Phase 6.)

## Error codes
- -32602 Invalid params — unknown/expired taskId (MUST for tasks/get,
  SHOULD for tasks/update and tasks/cancel)
- -32603 Internal error
- -32003 Missing Required Client Capability — client didn't declare
  the extension but requested task-aware behavior

## Optional (post-MVP) mechanisms — not required for v0.1.0
- notifications/tasks — server-pushed status updates
- subscriptions/listen — client opts into notifications/tasks per taskId
These are additive on top of polling; polling alone is spec-compliant.

## Deliberately out of scope for our TaskStore's public (wire) surface
- Any form of task enumeration/listing over MCP itself. An internal
  List() on the TaskStore interface is fine for the HTMX dashboard
  (a different, self-hosted-operator trust boundary) but must never
  be reachable via tasks/*.