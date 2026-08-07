# Audit User and Host Filter Selects

## Goal

Replace manual user and host inputs in audit-data filters with searchable dropdowns populated from the existing user-audit and host-audit statistics endpoints.

## Scope

The change applies to filters that query audit data:

- Audit event investigation: host, login user, and execution user.
- Command audit: user and host.
- Risk events: user and host.
- User audit: the page-level host filter and the detail host filter.
- Host audit: the detail user filter.
- Collector debug audit events: host.

The change does not apply to login, user management, host asset editing, rule-condition configuration, collector configuration, or system operation-log administrator fields. Those fields capture configuration or identities outside the audit user/host datasets.

## Design

Create shared audit-option utilities and select components for users and hosts. The components use Ant Design `Select`, preserve the existing form field values, support clearing and local searching, and expose loading and failure states.

User options come from `GET /stats/users` through `getUserAudits`. Each option uses `username` as both its submitted value and label. Empty usernames are removed and duplicate values are collapsed.

Host options come from `GET /stats/hosts` through `getHostAudits`. Each option submits the first available stable identity in this order: `hostId`, `nodeName`, then `hostName`. Its label presents the available host name, node name, and Host ID without repeating identical values. Empty identities and duplicate submitted values are removed.

Each page supplies its current time range to the shared component. The component requests up to the backend maximum of 100 audit records. Options refresh when the effective time range changes. Detail filters constrained to a selected user or host may continue using their already-loaded distribution data when that data is more specific than the global audit list, but they must use the same dropdown behavior and must not permit free-text entry.

## Data Flow

1. The page renders its filter form with the default or selected time range.
2. The shared selector loads the corresponding audit statistics for that range with `limit=100`.
3. Pure mapping helpers normalize, deduplicate, and label the returned records.
4. Selecting an option writes the existing string field into the form or detail-filter state.
5. Existing query builders send that value using their current query parameter names; backend filtering behavior remains unchanged.

## Interaction and Errors

- Dropdowns are searchable by their rendered labels and can be cleared.
- Loading is shown using the select's loading state.
- An empty response shows an explicit no-data message.
- A failed request shows an option-loading failure message and leaves the control non-editable; it does not fall back to manual text entry.
- Existing selected values remain stable during an option refresh.
- Requests are guarded against stale responses so a slower earlier range request cannot replace newer options.

## Testing

Follow test-driven development for the new pure option helpers:

- User options remove empty and duplicate usernames.
- Host options choose `hostId`, then `nodeName`, then `hostName` as the submitted value.
- Host labels contain all distinct available identities without duplication.

Because the frontend currently has no component-test runner, pure behavior tests will use the repository's TypeScript/build capabilities where practical; otherwise the helpers will be type-checked and the affected pages will be verified through a production build. Existing backend tests will be run to confirm the reused statistics endpoints and query limits remain valid. Browser verification will cover each affected dropdown, loading/empty behavior, searching, selection, clearing, and responsive layout.

## Non-Goals

- Adding a new backend options endpoint.
- Increasing the backend limit above 100.
- Allowing arbitrary values through Ant Design `Select` tags mode.
- Changing audit query semantics or the distinction between login and execution user fields.
