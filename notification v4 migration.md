# Notification V4 migration

Phase 1 storage and adapter changes passed the local rollout and rollback test. Verification with the release images and deployment to all affected processes remain open. Phase 2 is a proposal.

## 1. Objective and boundaries

Make the existing Notification database and its current service, adapter, consumers, and reconciler compatible with V4 before moving V4 onto shared storage. V4 will ultimately serve the V1, V3, and V4 HTTP APIs through version-specific translations.

The temporary `notification_v4*` tables isolate development today. They are not the intended production destination. Compatibility backports and schema consolidation remain prerequisites for merging the V4 change set. Preserve resource IDs, namespaces, timestamps, historical payloads, delivery histories, and existing Svix applications/endpoints.

This document expands the backport checklist in [NotificationV3.md](NotificationV3.md). It separates requirements supported by the implemented V4 service/adapter from additional persistence needed by the future reconciler. Registry implementation, domain-worker extraction, and the full event-driven reconciler remain separate workstreams; their integration dependencies are called out here.

### Two-phase rollout

**Phase 1 — deploy compatibility.** Add the schema and code support needed for the transition while old and new backends coexist. Keep stored values, provider routing, and webhook payloads readable and usable by old versions. Use temporary mappings, fallback reads, or additive projections where necessary. Existing representations remain authoritative until their replacement can be maintained by every writer. Do not enable new resource types or behavior that old readers or workers cannot handle.

**Phase 2 — finalize and remove compatibility.** After all affected API servers, workers, and scheduled jobs run the compatible release, make it the oldest supported rollback target. Switch writers and provider behavior to the target contracts, migrate existing data where needed, verify convergence, and remove temporary code and obsolete schema. V1 HTTP compatibility remains a supported contract; it is not temporary migration code.

**Any data migration that affects existing functionality belongs to Phase 2.** This includes changing routing, eligibility, delivery state, deduplication behavior, or values and relationships used by existing readers and writers, even if the SQL change is additive. Phase 1 may populate new fields or tables only when they leave existing functionality unchanged; an additive schema alone does not establish that guarantee.

These are two phases, not necessarily two deployments. Within Phase 2, keep fallback readers until every writer uses the final representation and backfills have caught up. Only then deploy cleanup. A rolling deployment of final writers must not overlap with readers that have already dropped support for values still being written by the previous release.

Every change proposal must identify its Phase 1 behavior, final representation, temporary measures, and removal conditions. Mark temporary code with `TODO(notification-v4)` and track its cleanup here.

For Channel types, Phase 1 is the current read-both/write-`WEBHOOK` adapter, with no data conversion. Phase 2 switches writes to `webhook`, converts remaining uppercase rows, and removes the legacy storage mapping after no uppercase writers or rows remain. Configuration envelope conversion has its own compatibility prerequisite; accepting both type discriminators does not imply support for the final raw configuration shape.

## 2. Current storage versus the target

The comparison below comes from the [legacy schema](openmeter/ent/schema/notification.go), [temporary V4 schema](openmeter/ent/schema/notificationv4.go), and [generated table definitions](openmeter/ent/db/migrate/schema.go). Existing ID, namespace, metadata, annotation, and lifecycle fields should retain their meaning. Physical SQL changes must be checked against the deployed schema; an Ent enum declaration alone is not proof of a PostgreSQL native enum.

The tables describe the final destination. In-place representation changes and removal of legacy fields belong to Phase 2. Phase 1 may populate additive fields or tables only when doing so preserves old behavior; projections must be caught up and verified before becoming authoritative.

### Channels: `notification_channels`

| Field or relation | Current representation | Required shared representation and backfill |
| --- | --- | --- |
| `type` | Ent enum, `WEBHOOK` | Ent string with lowercase identity `webhook`. Phase 1 reads both spellings, matches filters against both, and writes `WEBHOOK`. Phase 2 switches writes, converts remaining rows, and retires the uppercase storage mapping. |
| `config` | JSONB through a typed scanner: `{type, data}`; webhook fields include `customHeaders` and `signingSecret` | JSONB containing the configuration object directly: `url`, `custom_headers`, `signing_secret`. Convert envelopes and field names without rotating secrets or dropping headers. |
| `event_type_filter` | Absent | Optional JSONB string array, matching the V4 schema. Backfill the distinct canonical types of attached Rules, including disabled Rules. No wildcard values. |
| `provider_data` | Absent; some provider identifiers are annotations | Optional JSONB `map[string]string`. Adopt existing Svix identifiers using explicit key mappings and remote verification. Keep user metadata and system annotations separate. |
| `generation` | Absent | Go `int` / PostgreSQL `integer`, default `1`, range `1..MaxInt32`. Activated desired-state updates increment it; no observed generation is stored. |
| `reconcile_status` | Absent | String: `unknown`, `pending`, `in_sync`, or `error`; default `unknown`. |
| `reconcile_next_at` | Absent | Nullable timestamp for scheduled retries or checks; no default. |
| Rule assignments | `notification_channel_rules` join table | Retain during transition and rollback. Replace routing and compatibility reads with Event-type matching before retiring this relation. |

An empty filter means **catch-all**, including future Event types. It does not mean “no subscriptions.” A legacy Channel with no attached Rules must therefore be migrated as disabled, as agreed in the design, and recorded for operator review. Do not silently turn it into an enabled catch-all. Preserve existing soft-deleted state; do not recreate deleted endpoints.

Example configuration conversion:

```json
{"type":"WEBHOOK","data":{"url":"https://example.com/webhook","customHeaders":{"X-Team":"billing"},"signingSecret":"<existing-secret>"}}
```

```json
{"url":"https://example.com/webhook","custom_headers":{"X-Team":"billing"},"signing_secret":"<existing-secret>"}
```

The filter schema comment currently mentions patterns; align it with the implemented resolved-only contract when consolidating the schemas.

Phase 1 exposes config as `json.RawMessage` in Ent and decodes it in the adapter using the Channel's `type`. The adapter reads both the legacy envelope (with either discriminator spelling) and flat config, but continues writing `WEBHOOK` in the existing `{type, data}` envelope with camelCase keys. Reads do not convert rows. Phase 2 switches writes to flat config, converts stored rows, and removes envelope support after all writers and rows have converged.

Phase 1 adds `event_type_filter` as nullable JSONB without a default or backfill. Current operations leave it unset on creation and preserve stored values on updates. There are no dual writes or routing changes: Rule assignments remain authoritative. Phase 2 populates the filter before activating Event-type routing; an unset or empty filter does not activate catch-all behavior during Phase 1.

TODO(notification-v4): enforce unique filter entries in the V4 adapter when filter writes are implemented. The Phase 2 backfill must also produce distinct values. Do not add validation to the inactive Phase 1 field.

Phase 1 adds Channel `provider_data` as nullable JSONB without a default, dual writes, or backfill. Existing Channel operations leave it unset on creation and preserve it on updates, independently of metadata and annotations. Phase 2 adopts verified provider identifiers before using this field as the source of truth; current provider lookups remain unchanged in Phase 1.

Phase 1 adds Channel `generation` (Go `int` / PostgreSQL `integer`, default `1`, Ent range `1..MaxInt32`), `reconcile_status` (string, default `unknown`), and nullable `reconcile_next_at` (timestamp, no default). Allowed statuses are `unknown`, `pending`, `in_sync`, and `error`. Existing creation uses these inactive defaults; current updates and soft deletes preserve all three fields. No provider calls, scheduling, or HTTP behavior changes in this phase.

After activation, provider-relevant desired-state changes atomically increment `generation` and mark the Channel `pending`. The reconciler may mark `in_sync` only for the generation it processed, after synchronizing all managed fields. Detected provider drift marks `pending` without requiring a desired-state change. `reconcile_next_at` schedules retries or checks; normalized diagnostics remain in system annotations. No observed generation is stored. TODO(notification-v4): adopt actual provider state and enable tracking only after every writer supports it; never infer successful provisioning from migration defaults.

### Rules: `notification_rules`

| Field or relation | Current representation | Required shared representation and backfill |
| --- | --- | --- |
| `type` | Closed legacy Event-type enum | Canonical versioned string. Keep immutable after creation. Establish an explicit mapping for every existing type before converting rows. |
| `config` | JSONB `{type, data}` through the domain-specific Rule scanner | Opaque JSON configuration matching the registered version's schema. Conversion must preserve threshold operators, feature filters, and defaults; removing the wrapper alone is insufficient if the contract changes. |
| Channel/Event edges | Rules own Channel assignments and are required by Events | Stop requiring these edges in the new core. Preserve legacy associations while compatibility paths still need them. |

Deletion preserves config. Rule revisions are not required: the complete evaluated Rule snapshot, including `updated_at`, provides traceability without a separate counter. No revision column, increment, or revision validation is planned. Get by ID returns a soft-deleted Rule with its deletion timestamp. Multiple Rules with the same canonical Event type remain valid.

Phase 1 changes Rule `type` from an Ent enum to a string while retaining the existing `notification.EventType` validation and unversioned stored values. Type remains immutable, and config envelopes and Channel assignments stay unchanged. This step does not enable canonical versioned types or convert data; those require agreed contracts and compatibility mappings first.

Phase 1 exposes Rule `config` as `json.RawMessage` in Ent and moves the codec to the adapter. Reads try the legacy envelope first, then the unwrapped existing configuration using the Rule's type. Flat decoding rejects unknown fields, including envelope keys, so malformed envelopes do not become empty invoice configs. Writes keep the existing config validation and legacy envelope; reads never convert rows. Final versioned contracts, threshold changes, and data conversion remain separate work.

### Events: `notification_events`

| Field or relation | Current representation | Required shared representation and backfill |
| --- | --- | --- |
| `type` | Closed, unversioned Event-type enum | Canonical versioned string, consistent with Rule types, filters, dedupe scope, and Svix registration. |
| `payload` | JSONB exposed as a string containing a typed payload union | Opaque JSON via `json.RawMessage`; immutable in normal application writes. Retain historical meaning and numeric precision. Use explicit per-type conversion, not live domain lookups. |
| `rule_id` | Required FK to `notification_rules` | Make nullable for system Events; remove mandatory Rule loading. Retain existing values for V1 compatibility until their read/filter requirements have a replacement. Do not add a new hard Rule dependency to V4. |
| `rule_snapshot` | Absent | Optional JSONB snapshot of the evaluated Rule state, including its config and `updated_at`; no revision counter. Leave absent for historical Events unless the original evaluated state can be recovered reliably. The current Rule is not historical evidence. |
| `provider_data` | Absent | Optional JSONB string map, including the existing Svix message identity when available. |
| Dedupe aliases | Balance hash in annotations | New child table described below. Preserve unrelated annotations, including test markers and domain filtering attributes. |

Legacy types currently include `entitlements.balance.threshold`, `entitlements.reset`, `invoice.created`, and `invoice.updated`. The exact canonical names and versions must be agreed with domain owners. Do not automatically prefix every stored type with `v1.` and assume the payload contract matches. Invoice payloads have their own persisted `version: 1`; that is not automatically the new Event-type version.

Phase 1 changes Event `type` from an Ent enum to a string while retaining the existing `notification.EventType` validation and unversioned stored values. Type remains immutable; payloads, required Rule references, and routing are unchanged. Canonical versioned names and data conversion remain Phase 2 work.

Phase 1 exposes Event `payload` as `json.RawMessage` in Ent while keeping the JSONB column and stored JSON format unchanged. The adapter retains typed serialization, decoding, and invoice payload version checks. Reads do not rewrite historical payloads; raw storage preserves unknown fields and numeric precision. Final payload contracts and data conversion remain separate Phase 2 work.

Phase 1 makes Event `rule_id` nullable while preserving the foreign key and its `NO ACTION` deletion behavior. Current service and adapter creation paths still require an existing Rule and always write its ID. No existing references are cleared, and no Rule-free Events are introduced. Enabling Rule-free writes requires compatible readers, API mappings, routing, and reconciliation; this remains deferred. The down migration restores `NOT NULL` and is safe only while no null references exist.

Phase 1 adds nullable Event `rule_snapshot` JSONB storage, exposed as immutable `json.RawMessage` in Ent, without a default or backfill. Current creation paths leave it unset and existing operations preserve stored snapshots. API exposure and producer-supplied snapshots remain deferred. The future snapshot includes evaluated Rule state, config, and `updated_at`, with no revision counter; never populate historical snapshots from a live Rule.

Phase 1 adds Event `provider_data` as nullable JSONB `map[string]string` storage without a default, dual writes, or backfill. Current creation paths leave it unset and existing operations preserve populated values independently of annotations. Existing provider identities remain authoritative; adopting verified identities into this field and exposing it to reconciliation are deferred. No API or provider behavior changes in this step.

### Dedupe aliases: new `notification_event_dedupe_hashes` table

Phase 1 keeps the existing balance-hash annotation reads and writes. In Phase 2, create the shared `notification_event_dedupe_hashes` table, backfill verified aliases, and switch lookups after validating coverage and ownership. The temporary V4 equivalent is `notification_v4event_dedupe_hashes`.

| Column/constraint | Requirement |
| --- | --- |
| `id`, `namespace` | Existing ID/namespace conventions. |
| `event_id` | Immutable FK to Event ID; cascade on Event deletion. |
| `event_type`, `hash` | Immutable nonempty strings. Store the canonical type and the original hash value. |
| Unique `(namespace, event_type, hash)` | One owner per alias within its scope; an Event may own multiple aliases. |
| Index `event_id` | Efficient Event lookup and deletion. |

Use an ordinary Event-ID FK, consistent with the Ent constraint discussed previously. The adapter must verify that namespace and Event type match the owning Event; that consistency is not enforced by this FK alone.

**Backfill correction:** the [current balance handler](openmeter/notification/consumer/entitlementbalancethreshold.go) queries both V1 and V2 hashes but writes `dedupHash.V2()` into the scalar `event.balance.dedupe.hash` annotation for new Events. Do not assume each historical row contains two hashes. Import the hash actually present; add other aliases only when their original inputs can be recovered and recomputation is verified.

Audit duplicate ownership before completing the backfill. A conflict between different Events requires explicit resolution; do not merge Events or discard hashes automatically. Re-running the backfill must be harmless for aliases already owned by the same Event.

### Delivery statuses: `notification_event_delivery_status`

| Field or relation | Current representation | Required shared representation and backfill |
| --- | --- | --- |
| `state` | Ent enum: `PENDING`, `SENDING`, `RESENDING`, `SUCCESS`, `FAILED` | Lowercase string values with matching validation, queries, defaults, and worker switches. Translate V1 wire values at its boundary. |
| `dispatch_generation` | Absent | Go `int` / PostgreSQL `integer`, validated within `1..MaxInt32`, initially `1` at migration baseline. Explicit resend alone increments it. This baseline does not claim to reconstruct previous resend counts. Validate against PostgreSQL's supported integer range. |
| `dispatch_queued_at` | Absent from legacy writes | Nullable timestamp recording when the current dispatch generation was queued. Phase 1 adds storage without a default or backfill. |
| `attempts` | JSON containing legacy typed attempts/responses | JSONB array of V4 attempts with lowercase state, timestamp, opaque JSON response, and optional provider data. Explicitly map serialized fields and response units; do not directly cast the old Go type. |
| `provider_data` | Absent; Svix details occur in annotations | Optional JSONB string map. Populate verified provider identities and observations needed by resend recovery. |
| `event_id`, `channel_id` | Required string columns, without direct FKs to their named resources | Direct Event and Channel FKs using the established ID representation. Audit ownership and orphan rows before adding constraints. Event deletion cascades statuses; Channel deletion stays soft. |
| Event relation | Separate `notification_event_delivery_status_events` join table | Direct Event FK becomes authoritative. Verify the join agrees with `event_id`, maintain it while old readers use it, then remove it after those readers migrate. |
| Event–Channel index | Nonunique `(namespace, event_id, channel_id)` | Unique constraint on that tuple. Audit duplicate pairs and reconcile histories explicitly before enforcement. |
| `next_attempt_at` | Nullable timestamp plus `(namespace, state, next_attempt_at)` index | Retain while the current poller depends on it. The current V4 schema has no equivalent; do not lose scheduling state by copying that schema verbatim. |

Phase 1 uses a string field with `EventDeliveryStatusState` validation, uppercase writes, and the `PENDING` default. The adapter maps the five lowercase stored states to their uppercase domain values on reads. Delivery and Event queries match both spellings, including terminal-state exclusion for `next_attempt_at` scheduling. Phase 2 switches writers and converts stored states before retiring the compatibility mapping.

Phase 1 exposes delivery `attempts` as `json.RawMessage` in Ent and moves the existing typed codec to the adapter. The column remains JSONB; writes retain the legacy array, uppercase states, response fields, and nanosecond durations. Reads preserve SQL NULL, JSON null, and empty arrays without rewriting stored data; updates retain the existing nil-to-JSON-null behavior. Invalid attempt JSON fails both delivery and nested Event reads. V4 response normalization and data conversion remain Phase 2 work.

Phase 1 adds delivery-status `provider_data` as nullable JSONB `map[string]string` storage without a default, backfill, or dual writes. Current creation leaves it unset; state, attempt, scheduling, and annotation updates preserve stored values. Existing provider identities and reconciliation behavior remain authoritative. Provider adoption and API exposure are deferred.

Phase 1 adds delivery `dispatch_generation` with database and Ent default `1`, using Go `int` and PostgreSQL `integer`. Ent create/update validation requires `1..MaxInt32`. Existing rows receive the same baseline; it does not reconstruct prior sends. Current state, attempt, and resend updates preserve the field without incrementing it. TODO(notification-v4): activate guarded resend increments and provider adoption only after every writer can maintain the generation; it is not authoritative during mixed-version operation.

The relationship audit and any resulting repair or constraint enforcement are deferred to Phase 2. The legacy join table's cascade deletes relation rows, not the delivery-status aggregate. Direct Event-to-status cascade is a real behavior change and needs deletion tests. Validate that join rows, scalar IDs, namespaces, and Channel existence agree before consolidation.

Keep attempts together for now. A separate attempts table remains an open question for the end of implementation. Historical response JSON can be preserved as opaque data; any normalization to the new Svix response shape must explicitly convert legacy duration values to `duration_ms` and preserve V1 response compatibility. Do not fabricate provider attempt IDs from timestamps.

### Reconciler persistence still to finalize

Phase 1 preserves `dispatch_queued_at` across delivery updates and explicit resends. Historical deliveries have no reliable queue timestamp.

After activation, queueing an initial delivery sets `dispatch_queued_at`. An accepted explicit resend sets a fresh timestamp in the same transaction that advances the selected destination’s generation. Waiting for the Channel to reach `in_sync` counts toward `delivery_timeout`. Provider retries, callbacks, and observations preserve the original deadline for that generation.

`next_attempt_at` continues to represent the next delivery attempt for the poller and HTTP APIs. If reconciliation needs a separately scheduled observation, decide its storage with the reconciler. Recovery must resume outstanding Channel work across restart and gate native routing until the relevant endpoints converge.

Provider namespaces also need an ownership decision: V4 exposes namespace management and `svix.application_id`, but neither Notification schema currently defines a namespace resource table. Reuse the existing namespace lifecycle or choose its persistence owner; do not introduce a Notification namespace table implicitly in this migration.

## 3. Backports to the current service and adapter

### Shared readers, writers, and HTTP translation

Update the [legacy mappers](openmeter/notification/adapter/entitymapping.go), Ent scanners, validators, and query predicates before changing stored representations. Phase 1 readers must understand both old and new forms, while writers retain the old representation. Writers must preserve fields they do not own, particularly provider data, annotations, filters, generation, and attempt history. New fields that old writers cannot maintain must not become authoritative until Phase 2 catches up their values and retires those writers.

The generated Ent code, migration SQL, and schema checks must move together. Retain namespace scoping, caller-owned transactions, and row locking. Repoint the [V4 adapter](openmeter/notification/v4/adapter/adapter.go) to shared entities only after backports and backfills are verified; remove temporary generated entities at the end. If temporary V4 rows exist outside disposable tests, copy them with preserved IDs and explicit collision checks.

Preserve existing query indexes until their consumers migrate. Plan unique-index creation, FK validation, and column conversions against actual table sizes and lock budgets. Run data backfills in bounded, restartable batches with progress and exception counts; coordinate concurrent writes so an older backfill snapshot cannot overwrite newer config or provider state. Do not trigger resource lifecycle events or provider mutations merely because a row's storage representation changed.

The [HTTP mappings](openmeter/notification/httpdriver/mapping.go) must continue to produce V1 response shapes and enum values. Keep existing pagination, filter behavior, error mappings, and payload conversions where promised. New internal exact-type filters need translation from legacy type names. Future wildcard HTTP support must resolve before service calls and must never convert an unmatched pattern into an empty catch-all Channel filter.

Define how V1 handles types and system Events its response models cannot represent, including direct GET as well as list. Do not allow one unsupported record to make a whole list fail. V4 no longer carries a live Rule ID on Events, whereas V1 reads and filters use it; retaining nullable `rule_id` during compatibility is necessary but not sufficient without an explicit read/write bridge.

### Channel and Rule operations

- [Channel service](openmeter/notification/service/channel.go): preserve filters/provider data on all updates; convert config without losing omitted fields; retain signing secrets on adoption; support deleted-resource reads. Remove the Rule-reference deletion restriction only when routing no longer depends on it.
- [Rule service](openmeter/notification/service/rule.go): validate the merged state before persistence; no revision counter or increment check is required. Preserve immutable Event type and deleted reads. Integrate Registry validation where required once its implementation is available; do not add a permissive production fallback.
- Before routing cutover, continue maintaining the old assignments and provider routing while projecting their canonical Event types into Channel filters. After cutover, V1 Rule writes **add** the type to selected Channels; unassigning or deleting a Rule does not remove the type. Rule reads derive matching Channels, including catch-all Channels. Only Channel updates remove subscriptions.
- Keep transaction coordination and resource-change publishing in the service. Publish wakeups after the outer commit; a failed publish leaves durable work for recovery polling. The current synchronous provider calls must be removed or gated when asynchronous Channel reconciliation becomes authoritative.

There is an intentional semantic change: two Rules with the same Event type can now deliver to the same subscribed Channel even if only one was previously assigned. Preserve V1 wire compatibility, but do not describe the routing transition as behavior-identical. Report this expansion during migration review.

### Event creation and deduplication

Phase 1 keeps the current [service](openmeter/notification/service/event.go) and [adapter](openmeter/notification/adapter/event.go) creation path. The service requires an enabled live Rule. Within the same transaction, the adapter persists the Event and creates a `PENDING` delivery status for each active Channel assigned to that Rule.

**Explicit destination selection is deferred until V4 Event creation.** The V4 creation contract requires the following:

1. Accept Rule-free Events while keeping legacy producer checks in their compatibility path.
2. Have the service select destinations and pass explicit Channel IDs to the adapter. For example, a system Event can be persisted with zero matching Channels or with a delivery status for each match. Activate Event-type routing at the coordinated cutover.
3. Atomically persist Event, dedupe aliases, and initial statuses. Empty destination sets remain valid. A duplicate Event returns its existing destinations/history rather than recomputing them.
4. Move alias lookup/ownership enforcement to the dedicated table. On a single matching Event, compare content and add missing aliases; multiple owners or different content are conflicts. Concurrent alias conflicts roll back before a fresh lookup.
5. Preserve JSON numbers during content comparison and annotation reads; round-tripping through `map[string]any` can otherwise change large numeric values.

Adapt all [consumers](openmeter/notification/consumer), not only balance thresholds. Entitlement reset has a different legacy deduplication lookup; invoice handling and test-event generation also need the shared boundary. Preserve existing evaluation and payload snapshots while domain-owned evaluation workers are introduced separately.

V4's structured identity hashes (`event:v1:...`) are not the existing balance hashes. Supply a verified alias bridge during producer transition; changing algorithms without shared aliases can create duplicate Events. Keep annotation fallback/dual writes only for a bounded rollout period, then remove them after compatible writers and backfill coverage are confirmed. Do not change existing hash bytes or assume a new prefix makes algorithms equivalent.

### Delivery updates and resend

Backport unique-pair conflicts, partial updates, opaque attempt responses, and provider-data persistence to the [delivery adapter](openmeter/notification/adapter/deliverystatus.go). Omitted fields preserve state; explicit empty collections clear them. Preserve other providers' keys when applying a provider result.

Resend keeps the same Event and statuses. Lock the affected resources in a consistent order, re-read eligibility, reject the whole request if any selected destination is active/ineligible, then increment only selected generations and set `resending` atomically. Do not clear previous attempts or provider data. The legacy service currently permits `SENDING`; moving to terminal-only resend is an intentional behavior change that must be covered in V1 compatibility tests and rollout communication.

Event history queries must use persisted Event–Channel relationships. The current Channel filter traverses the Event's live Rule assignments; that cannot work for system Events or reliably describe historical destinations after rules change.

## 4. Svix and current reconciler changes

The [current reconciler](openmeter/notification/eventhandler/reconcile.go) uses `next_attempt_at` to select active deliveries. The adapter matches both stored state spellings. The reconciler still discovers channel types through `event.Rule.Channels`. The [webhook reconciler](openmeter/notification/eventhandler/webhook.go) sends the Rule ID as a Svix channel, checks endpoint Rule membership, and converts typed payloads into legacy webhook envelopes. Adopting V4 resources also requires changes to routing and payload handling.

### Adopt existing remote resources

Keep namespace ID as Svix application UID, Channel ID as endpoint UID, and Event ID as message external ID. Read existing resources and persist their actual provider identifiers instead of recreating them.

| Existing provider annotation | V4 provider data | Migration action |
| --- | --- | --- |
| `svix.application.id` (if present) | `svix.application_id` | Verify during the [Phase 2 routing cutover](#coordinated-routing-cutover). Namespace provider-data ownership is still open. |
| `svix.endpoint.id` | `svix.endpoint_id` on Channel | Verify against endpoint UID; preserve URL, headers, signing secret, disabled state, and unrelated remote settings. |
| `svix.message.id` | `svix.message_id` on Event | Existing message annotations may be on statuses. Check consistency across destinations and verify when possible. |
| No equivalent reliable attempt identity | `svix.attempt_id` on attempt | Obtain from provider observations when available; leave historical unknown IDs unknown. |

These are explicit translations, not a blanket annotation copy. Keep compatibility annotations until old consumers stop using them. Missing/expired remote messages are exceptions to classify, not permission to send a historical Event again.

Before enabling V4 resend on an adopted delivery, observe and persist its generation-1 `svix.observed_attempts` and `svix.dispatch_generation`. V4's [resend recovery](openmeter/notification/v4/provider/svix/README.md) uses that persisted observation as the next generation's baseline. A generation number without the observation is insufficient. Drain or explicitly resolve legacy `RESENDING` work before adoption; legacy state cannot prove whether a remote resend was already accepted.

### Extend reconciliation before switching routing

- Read lowercase/canonical storage through compatible adapters, and observe destinations from persisted statuses and Channels rather than live Rules. Do not fail an adopted delivery merely because its Rule was deleted or unassigned.
- Add resumable Channel synchronization and repair, including Channels with no pending Events. The existing Event-only poller cannot discover every endpoint that needs migration.
- Persist provider results, partial destination outcomes, and resend observations. Recover already accepted messages by Event ID without resubmission; keep acceptance distinct from delivery success.
- Share resource locking and ownership across old polling, Kafka wakeups, callbacks, and migration repair. Only one path may mutate or dispatch a given resource. Kafka is a wakeup mechanism, with polling recovery; no outbox is required by this design.
- Preserve the current provider-disabled annotation/user re-enable behavior during transition. Prevent endpoint repair from repeatedly undoing a provider-side disable without an explicit policy.

The [V4 Svix Channel update](openmeter/notification/v4/provider/svix/channel.go) clears legacy Rule-ID filters when Event types are supplied. Calling it during shadow comparison would already switch routing. Shadow mode must be read/compare only for those fields until cutover is authorized.

### Coordinated routing cutover

Phase 1 keeps Rule assignments authoritative in OpenMeter and Svix. At the start of the Phase 2 routing cutover, compare the existing destinations with the proposed Event-type destinations for each cohort. Review differences before changing any endpoint.

Before changing routing for a namespace, call [`GetNamespace`](openmeter/notification/v4/provider/svix/namespace.go) with the namespace ID as the Svix application UID. Require the returned UID to match and the generated application ID to be present. Compare that ID with any recorded `svix.application.id` values. A missing annotation is acceptable when the lookup succeeds.

A 404 means the application is missing. A conflicting ID or UID means its identity is unresolved. Pause the affected cutover in either case. Retry transient lookup failures before proceeding. This check is read-only and runs when adopting the namespace in Phase 2. Calling `GetOrCreate` here could silently provision a replacement.

Current mapping: Notification Channel → Svix endpoint; Notification Rule → entry in endpoint `channels`. Target: endpoint Event-type filters with canonical types; initial sends use native Svix fan-out, not explicit endpoint IDs.

1. Register the canonical Svix Event types and prepare Channel filter projections. The V4 provider does not register Event types itself; assign that responsibility in application composition/Registry integration.
2. Compare old Rule destinations against projected Event-type destinations. Explain expected broadening and reject accidental catch-all, disabled-state, or missing-endpoint differences.
3. Determine a safe cutover cohort. Prove mixed old/new routing with the deployed Svix version before allowing per-Channel migration; otherwise coordinate the whole affected application or wider provider scope.
4. Fence Event creation/routing decisions and new submissions for that cohort, or queue work without fixing incompatible destinations. Drain or explicitly handle existing pending/in-flight/resending work. Events whose destinations were captured before the switch need a defined legacy completion path; do not silently add destinations to them.
5. Replace endpoint Rule-ID channels with canonical Event-type filters and read back the managed state. Catch-all uses cleared filters, never a placeholder Rule ID. Resume only once all relevant endpoints converge; native fan-out cannot honor a local destination list when remote filters differ.
6. Enable Event-type destination selection and canonical submission together. Keep recovery polling and a repair checkpoint until all resources are verified.

Webhook wire compatibility is a separate release decision: [legacy delivery](openmeter/notification/eventhandler/webhook.go) uses HTTP API payload mappers; [V4 delivery](openmeter/notification/v4/provider/svix/eventmapping.go) sends `{id, event_type, created_at, payload, annotations, rule_snapshot?}`. Reusing an endpoint does not make these envelopes compatible. Agree a compatibility renderer or explicit customer cutover before switching its messages. Historical targeted resends must continue to use the original provider message/payload.

## 5. Small implementation increments and merge gates

### Phase 1: deployable compatibility

| Order | Deliverable | Gate before the next step |
| --- | --- | --- |
| 1 | Identify legacy storage formats, webhook contracts, and the processes that read and write them. | Document the compatibility mapping and when each temporary measure can be removed. |
| 2 | Add nullable fields and safe defaults. Keep old relations, values, config shapes, and scheduling fields. | Old binaries can still read and write; no new constraint rejects a valid old write. |
| 3 | Compatible readers, legacy-format writers, service/adapter backports, and reconciler support. Populate additive projections where safe. | Old and new instances interoperate in both directions; projections are not trusted while old writers can make them stale. |
| 4 | Run the rollout and rollback smoke test with release images, then deploy the compatible release. | Verify the running API servers and workers and the image configured for scheduled jobs. Record the compatible release as the oldest supported rollback target before starting Phase 2. |

**Release verification.** Run the [smoke script](test/notification/rolling/smoke.sh) from the repository root with the deployed and candidate image digests. Set `OLD_JOBS_IMAGE` and `CANDIDATE_JOBS_IMAGE` when jobs are packaged separately. Keep the run log and image references with the release record.

```sh
OLD_IMAGE="$deployed_image_digest_ref" CANDIDATE_IMAGE="$candidate_image_digest_ref" \
  test/notification/rolling/smoke.sh
```

Two local runs passed against the candidate built from `bd4cab68b`. One used the pre-Phase-1 commit `ee44d4c99` as its baseline. The other used the published [v1.0.0-beta.236 release](https://github.com/openmeterio/openmeter/releases/tag/v1.0.0-beta.236), pinned to `ghcr.io/openmeterio/openmeter@sha256:7d6982a0c12cce366593740f61d9f53960e2ac9af20be2ce011b8c154ee5ebb7`.

In the release run, PostgreSQL advanced from `20260925133042` to `20261109132325`. The candidate preserved the pending delivery and wrote legacy-readable Channel, Rule, and Event records. After rollback on the same upgraded database, the released server delivered the candidate Event. Resend increased its persisted attempt count from one to two and produced a second matching webhook.

### Phase 2: finalization and cleanup

**Adopting in-flight deliveries.** Drain all writers that can queue or resend a delivery without recording its queue time. Capture one UTC cutover timestamp and persist it in the migration checkpoint. Every batch and restart must reuse that timestamp.

Set `dispatch_queued_at` to the cutover timestamp only on active deliveries (`PENDING`, `SENDING`, `RESENDING`, or their lowercase equivalents) where it is still NULL. Check both conditions in the update so a delivery completed or requeued concurrently keeps its current values. Leave terminal deliveries and existing timestamps untouched. Preserve `next_attempt_at`, dispatch generation, state, and attempts. Adoption itself never sends or resends an Event.

Enable timeout enforcement once every active delivery has a queue timestamp. Adopted deliveries expire at `cutover timestamp + delivery_timeout`, including time spent waiting for the Channel to reach `in_sync`. For a 10:00 UTC cutover and a 30-minute timeout, the deadline is 10:30 UTC even if a batch resumes at 10:10.

| Order | Deliverable | Gate before the next step |
| --- | --- | --- |
| 5 | Create the dedupe alias table before deploying writers that use it. Run resumable conversions and catch up aliases, direct FKs, filters, and other projections. Add final constraints after validation. | All writers use the target format; counts, content checks, relation parity, and repeated-run checks pass. Fallback readers remain during convergence. |
| 6 | Consolidate V4 onto shared entities and complete cross-version tests; retire temporary development tables after verification. | V1 support and shared-schema consolidation are complete before merging V4. New production behavior stays gated until routing is ready. |
| 7 | Coordinated Svix routing and service cutover, then staged event-driven reconciliation. | Verify each namespace’s Svix application identity before switching routing. Fence/drain checks, endpoint readback, delivery observation, and recovery tests pass for each cohort. |
| 8 | Remove temporary mappings, fallback reads/writes, obsolete joins/fields, annotation dedupe, and migration TODOs. | No active or supported rollback version, stored row, queued work, or historical API requirement depends on them. Keep permanent V1 API translations. |

Phase 2 starts only after the compatibility deployment is complete. The compatible release becomes the rollback floor; an unmodified legacy binary is no longer a safe rollback once final storage or behavior is activated. Keep DB and remote-state repair records per cutover cohort. Rolling back SQL does not undo Svix changes or delivered webhooks, and cleanup must not remove data the supported rollback release still needs.

## 6. Verification and unresolved decisions

Migration tests should start from real legacy-format fixtures and cover all four Event types, old/new hash forms, deleted resources, disabled/unassigned Channels, duplicate alias owners, duplicate delivery pairs, broken join relations, and in-flight/resending work. Verify stable IDs/timestamps, JSON numeric precision, secret/header preservation, resumable backfill, transaction rollback, and direct-FK cascade cleanup.

For Phase 1, verify both directions: old writes remain readable and usable by new code, and new writes remain readable and usable by old code. For Phase 2, cover mixed compatible/final writers, backfill catch-up, rollback to the supported compatible release, and cleanup only after legacy representations and dependencies are gone.

Run the current service/adapter/consumer regression suites against the shared schema alongside V4 PostgreSQL BDD tests with `-race`. Add live Svix cases for existing endpoint adoption, both routing modes, catch-all, filters cleared on update, unchanged webhook contracts, accepted-message recovery, missing remote history, and resend after an ambiguous response. Verify publication failure still leaves recoverable work and that two worker paths cannot dispatch it independently.

The following decisions are needed before implementation/cutover, not new defaults established by this document:

- **Canonical contracts:** exact per-type names/versions, Rule config conversion, and historical payload mapping. Connect generated contracts to the deferred Registry implementation before enabling producers that require it.
- **V1 visibility and Rule identity:** representation of system/unknown Events, legacy Rule-ID filters, and preservation of V1 webhook payloads. The current V4 model alone cannot supply all of this information.
- **Routing transition:** cohort size, pending-work handling, and acceptance of same-type Rule routing broadening. Validate Svix behavior rather than assuming endpoint filters form an atomic migration across resources.
- **Operational persistence:** Activation of Channel reconciliation tracking, delivery scheduling/deadlines, namespace provider-data ownership, and the checkpoint mechanism for resumable repair.
- **Historical resend:** handling missing provider history and ambiguous legacy resends; do not invent an empty baseline or silently dispatch a new message.

Retention is separate from migration activation. The agreed policy is initially 90 days from Event creation, deleting only when all destinations are terminal and no work remains. Establish the Event→status/alias cascades in Phase 2 before enabling bounded cleanup. Expired aliases end the corresponding deduplication guarantee. Retain the current attempts JSON design until the recorded performance question is revisited.
