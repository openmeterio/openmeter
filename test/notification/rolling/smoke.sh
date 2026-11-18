#!/usr/bin/env bash
# Requires Bash, Docker Compose, curl and jq. From the repository root:
# OLD_IMAGE=<beta.237 image> CANDIDATE_IMAGE=<candidate> test/notification/rolling/smoke.sh
# Images contain both binaries. Override OLD_JOBS_IMAGE/CANDIDATE_JOBS_IMAGE for split builds.
set -euo pipefail

: "${OLD_IMAGE:?provide the baseline image}" "${CANDIDATE_IMAGE:?provide the candidate image}"
cd "$(dirname "$0")"
artifacts=$(mktemp -d "${TMPDIR:-/tmp}/notification-rollout.XXXXXXXX")
project="notification-rollout-$(date +%s)-$$"
export RECONCILE_INTERVAL=1h
dc() { docker compose --project-name "$project" -f compose.yaml "$@"; }

cleanup() {
    local rc=$?
    trap - EXIT
    dc logs --no-color >"$artifacts/compose.log" 2>&1 || true
    dc down --volumes --remove-orphans >>"$artifacts/compose.log" 2>&1 || rc=1
    echo "Smoke artifacts: $artifacts"
    exit "$rc"
}
trap cleanup EXIT

fail() { echo "$*" >&2; exit 1; }
sql() { dc exec -T postgres psql -XqAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
version() { sql 'SELECT version FROM schema_om WHERE NOT dirty'; }
identity() { sql "SELECT system_identifier || '/' || oid FROM pg_control_system(), pg_database WHERE datname=current_database()"; }

start_server() {
    local role=$1 port url
    dc up -d "$role"
    port=$(dc port "$role" 8888)
    url="http://$port/api/v1/notification"
    case "$role" in
        old) old_url=$url ;;
        candidate) candidate_url=$url ;;
    esac

    local deadline=$((SECONDS + 150))
    until curl -fsS --max-time 3 "$url/channels" >/dev/null 2>&1; do
        (( SECONDS < deadline )) || fail "$role API did not become ready"
        sleep 1
    done
}

api() {
    local role=$1 method=$2 path=$3
    shift 3
    local base="${role}_url"
    curl --fail-with-body -sS --max-time 30 -X "$method" -H 'Content-Type: application/json' "${!base}/$path" "$@"
}

id_from() { jq -er '.id | select(test("^[0-9A-HJKMNP-TV-Z]{26}$"))' "$1"; }
seed() {
    local role=$1 dir="$artifacts/$1" channel rule event
    mkdir "$dir"
    jq -n --arg role "$role" '{type:"WEBHOOK", name:$role, url:("http://receiver:8080/"+$role),
        customHeaders:{"X-Smoke":$role}, signingSecret:"whsec_MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3"}' >"$dir/channel-input"
    api "$role" POST channels --data-binary "@$dir/channel-input" >"$dir/channel"
    channel=$(id_from "$dir/channel")

    jq -n --arg role "$role" --arg id "$channel" '{type:"entitlements.reset",name:$role,channels:[$id]}' >"$dir/rule-input"
    api "$role" POST rules --data-binary "@$dir/rule-input" >"$dir/rule"
    rule=$(id_from "$dir/rule")

    api "$role" POST "rules/$rule/test" >"$dir/created-event"
    event=$(id_from "$dir/created-event")
    # GET observes PostgreSQL timestamp precision, just as the reconciler does.
    api "$role" GET "events/$event" >"$dir/event"
    jq -e --slurpfile created "$dir/created-event" '
        (.payload | {id,type,data}) == ($created[0].payload | {id,type,data})' "$dir/event" >/dev/null
}

cross_read() {
    local reader=$1 writer=$2 dir="$artifacts/$2" channel rule event
    channel=$(id_from "$dir/channel")
    rule=$(id_from "$dir/rule")
    event=$(id_from "$dir/event")
    api "$reader" GET "channels/$channel" | jq -e --slurpfile created "$dir/channel" '
        {id,type,url,customHeaders} == ($created[0] | {id,type,url,customHeaders})' >/dev/null || fail "$reader could not read $writer Channel"
    api "$reader" GET "rules/$rule" | jq -e --arg id "$rule" --arg c "$channel" '
        .id==$id and .type=="entitlements.reset" and [.channels[].id]==[$c]' >/dev/null || fail "$reader could not read $writer Rule"
    api "$reader" GET "events/$event" | jq -e --slurpfile created "$dir/event" --arg c "$channel" --arg r "$rule" '
        .rule.id==$r and .id==$created[0].id and .payload==$created[0].payload and
        [.deliveryStatus[].channel.id]==[$c]' >/dev/null || fail "$reader could not read $writer Event"
    echo "$reader read $writer Channel, Rule and Event"
}

stored_format() {
    local writer=$1 expected_channel_type=$2 dir="$artifacts/$1" event
    event=$(id_from "$dir/event")
    sql "SELECT jsonb_build_object('channel', c.type, 'config', c.config, 'rule', r.type, 'ruleConfig', r.config,
            'event', e.type, 'payload', e.payload, 'state', s.state)
         FROM notification_events e JOIN notification_rules r ON r.id=e.rule_id
         JOIN notification_event_delivery_status s ON s.event_id=e.id
         JOIN notification_channels c ON c.id=s.channel_id WHERE e.id='$event'" >"$dir/stored.json"
    jq -es --arg writer "$writer" --arg channel_type "$expected_channel_type" --slurpfile input "$dir/channel-input" --slurpfile rule_input "$dir/rule-input" 'length==1 and (.[0] |
        .channel==$channel_type and .config==($input[0] | if $writer=="old" then
            {type:"WEBHOOK",data:{url,customHeaders,signingSecret}}
        else {url,custom_headers:.customHeaders,signing_secret:.signingSecret} end) and
        .rule==(if $writer=="old" then "entitlements.reset" else "v1.entitlement.usage_period.reset" end) and
        .ruleConfig==({features:$rule_input[0].features} |
            if $writer=="old" then {type:"entitlements.reset",data:.} else . end) and
        .event==.rule and .payload.type=="entitlements.reset" and .payload.version==1 and
        (.payload.entitlementReset|type)=="object" and .state==(if $writer=="old" then "PENDING" else "pending" end))' "$dir/stored.json" >/dev/null ||
        fail "$writer stored an incompatible representation"
    echo "Verified $writer stored Channel, Rule, Event and pending delivery formats"
}

receipts() { dc exec -T receiver cat /tmp/receipts.ndjson; }
await_delivery() {
    local reader=$1 writer=$2 wanted=$3 previous=$4 dir="$artifacts/$2" event count deadline expected_state
    expected_state=SUCCESS
    [[ $reader != candidate ]] || expected_state=success
    event=$(id_from "$dir/event")
    deadline=$((SECONDS + 180))
    until (( SECONDS >= deadline )); do
        api "$reader" GET "events/$event" >"$dir/observed.json"
        count=$(sql "SELECT jsonb_array_length(attempts) FROM notification_event_delivery_status
            WHERE event_id='$event' AND state IN ('SUCCESS', 'success') AND jsonb_typeof(attempts)='array'")
        if [[ $count =~ ^[0-9]+$ ]] && (( count > previous )) &&
            jq -e --argjson n "$count" '.deliveryStatus | length==1 and
                (.[0] | .state=="SUCCESS" and (.attempts|length)==$n)' "$dir/observed.json" >/dev/null &&
            receipts | jq -se --arg path "/$writer" --argjson n "$wanted" --slurpfile e "$dir/event" '
                map(select(.body.id==$e[0].id)) | length==$n and all(.path==$path and .body==$e[0].payload)' >/dev/null; then
            [[ $(sql "SELECT state FROM notification_event_delivery_status WHERE event_id='$event'") == "$expected_state" ]] ||
                fail "$reader wrote an unexpected delivery state"
            echo "$reader reconciled $writer Event: $count stored attempts, $wanted matching receipts"
            return
        fi
        sleep 2
    done
    fail "Timed out waiting for $reader to reconcile $writer Event $event (attempt count must exceed $previous)"
}

# Each jobs image migrates once. Both APIs have autoMigrate disabled, including on rollback.
dc up -d --wait --wait-timeout 180 postgres redis kafka clickhouse svix receiver
dc run --rm -T old-jobs >"$artifacts/old-migration.log" 2>&1
db=$(identity)
baseline=$(version)
[[ $baseline =~ ^[0-9]+$ ]] || fail "Invalid baseline schema version: $baseline"
start_server old
seed old
old_event=$(id_from "$artifacts/old/event")
stored_format old WEBHOOK
sql "UPDATE notification_event_delivery_status SET next_attempt_at=now()-interval '1 minute' WHERE event_id='$old_event'"
pending=$(sql "SELECT jsonb_build_array(id,state,next_attempt_at) FROM notification_event_delivery_status WHERE event_id='$old_event'")

dc run --rm -T candidate-jobs >"$artifacts/candidate-migration.log" 2>&1
upgraded=$(version)
[[ $upgraded =~ ^[0-9]+$ ]] && (( upgraded >= baseline )) || fail "Expected schema version at least $baseline, got $upgraded"
[[ $(identity) == "$db" ]] || fail "Database changed during migration"
[[ $(sql "SELECT jsonb_build_array(id,state,next_attempt_at) FROM notification_event_delivery_status WHERE event_id='$old_event'") == "$pending" ]] ||
    fail "Migration changed the in-flight delivery"
echo "PostgreSQL $db migrated from $baseline to $upgraded"

start_server candidate
cross_read candidate old
seed candidate
event=$(id_from "$artifacts/candidate/event")
channel=$(id_from "$artifacts/candidate/channel")
stored_format candidate webhook
cross_read old candidate

# Defer this destination until rollback. A sentinel detects an old writer clearing an unfamiliar column.
sql "UPDATE notification_event_delivery_status SET next_attempt_at=now()+interval '1 day' WHERE event_id='$event'"
sql "UPDATE notification_channels SET provider_data='{\"smoke\":\"preserve\"}' WHERE id='$channel'"
jq '.name="updated by old"' "$artifacts/candidate/channel-input" >"$artifacts/old-update"
api old PUT "channels/$channel" --data-binary "@$artifacts/old-update" >/dev/null
[[ $(sql "SELECT name || '/' || type || '/' || (provider_data->>'smoke') FROM notification_channels WHERE id='$channel'") == 'updated by old/webhook/preserve' ]] ||
    fail "Old update changed the candidate Channel type or lost provider data"
sql "SELECT config FROM notification_channels WHERE id='$channel'" | jq -e --slurpfile input "$artifacts/old-update" '
    .=={type:"WEBHOOK",data:($input[0] | {url,customHeaders,signingSecret})}' >/dev/null ||
    fail "Old update did not preserve the candidate webhook settings in its envelope"
cross_read candidate candidate
echo "Old server updated candidate Channel and preserved type, webhook settings and provider_data"

rule=$(id_from "$artifacts/candidate/rule")
jq '.name="updated by old"' "$artifacts/candidate/rule-input" >"$artifacts/old-rule-update"
api old PUT "rules/$rule" --data-binary "@$artifacts/old-rule-update" >/dev/null
sql "SELECT jsonb_build_object('type',type,'config',config) FROM notification_rules WHERE id='$rule'" |
    jq -e --slurpfile input "$artifacts/old-rule-update" '
        .type=="v1.entitlement.usage_period.reset" and
        .config=={type:"entitlements.reset",data:{features:$input[0].features}}' >/dev/null ||
    fail "Old update changed the candidate Rule type or config values"
cross_read candidate candidate
echo "Old server updated candidate Rule and restored its legacy config envelope"


# Only one reconciler may deliver during each handoff.
dc stop old
export RECONCILE_INTERVAL=1s
start_server candidate
await_delivery candidate old 1 0
receipts | jq -se 'length==1' >/dev/null || fail "Candidate delivered the held Event"
dc stop candidate

start_server old
[[ $(identity) == "$db" && $(version) == "$upgraded" ]] || fail "Rollback changed the database or schema version"
cross_read old candidate
# The old reader must also decode the attempt written by the candidate during handoff.
api old GET "events/$old_event" | jq -e --slurpfile observed "$artifacts/old/observed.json" '
    .deliveryStatus== $observed[0].deliveryStatus and .deliveryStatus[0].attempts[0].response.statusCode==204' >/dev/null ||
    fail "Old server could not read candidate delivery attempts"
sql "UPDATE notification_event_delivery_status SET next_attempt_at=now()-interval '1 minute' WHERE event_id='$event'"
await_delivery old candidate 1 0
before=$(sql "SELECT jsonb_array_length(attempts) FROM notification_event_delivery_status WHERE event_id='$event'")
api old POST "events/$event/resend" --data "$(jq -n --arg c "$channel" '{channels:[$c]}')" >/dev/null
await_delivery old candidate 2 "$before"
receipts >"$artifacts/receipts.ndjson"
jq -se 'length==3' "$artifacts/receipts.ndjson" >/dev/null || fail "Unexpected webhook receipts"
echo "PASS: rollback to the old server on upgraded PostgreSQL $db (schema $upgraded)"
