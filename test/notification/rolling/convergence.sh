# Sourced by smoke.sh. Reapply the shipped data migrations after a binary rollback:
# the jobs migrator skips versions recorded as successful on the first upgrade.
backfills=(
    ../../../tools/migrate/migrations/20261124090000_notification_channel_type_normalization.up.sql
    ../../../tools/migrate/migrations/20261124090100_notification_channel_config_normalization.up.sql
    ../../../tools/migrate/migrations/20261124090200_notification_event_type_normalization.up.sql
    ../../../tools/migrate/migrations/20261124090300_notification_delivery_state_normalization.up.sql
    ../../../tools/migrate/migrations/20261124090400_notification_rule_config_normalization.up.sql
)

check_counts() {
    local stage=$1 expectation=$2 counts="$artifacts/$1-counts.json"
    local legacy_types="'entitlements.balance.threshold','entitlements.reset','invoice.created','invoice.updated'"
    sql "SELECT jsonb_build_object(
        'channel_type', (SELECT count(*) FROM notification_channels WHERE type='WEBHOOK'),
        'channel_config', (SELECT count(*) FROM notification_channels WHERE type IN ('WEBHOOK','webhook') AND config ?| ARRAY['type','data']),
        'rule_type', (SELECT count(*) FROM notification_rules WHERE type IN ($legacy_types)),
        'event_type', (SELECT count(*) FROM notification_events WHERE type IN ($legacy_types)),
        'delivery_state', (SELECT count(*) FROM notification_event_delivery_status WHERE state IN ('PENDING','SENDING','RESENDING','SUCCESS','FAILED')),
        'rule_config', (SELECT count(*) FROM notification_rules WHERE type IN ($legacy_types,
            'v1.entitlement.balance_threshold.triggered','v1.entitlement.usage_period.reset','v1.billing.invoice.created','v1.billing.invoice.updated')
            AND config ?| ARRAY['type','data']))" >"$counts"
    echo "$stage: $(cat "$counts")"
    jq -e "$expectation" "$counts" >/dev/null || fail "Unexpected legacy counts at $stage"
}

replay() {
    local migration
    for migration in "$@"; do
        dc exec -T postgres psql -Xq -U postgres -d postgres -v ON_ERROR_STOP=1 <"$migration"
    done
}

row_versions() {
    # A no-op UPDATE still changes xmin. Comparing every test row catches rewrites
    # even when the SQL assigns the same value that was already there.
    local table
    for table in notification_channels notification_rules notification_events notification_event_delivery_status; do
        sql "SELECT '$table', id, xmin FROM $table ORDER BY id"
    done
}

replay_during_rule_edit() {
    local id=$1 fifo="$artifacts/editor.pipe" editor migration deadline
    mkfifo "$fifo"
    dc exec -T -e PGAPPNAME=smoke_editor postgres psql -Xq -U postgres -d postgres -v ON_ERROR_STOP=1 \
        <"$fifo" >"$artifacts/editor.log" 2>&1 &
    editor=$!
    exec 3>"$fifo"
    unlink "$fifo"
    # A current writer replaces a legacy envelope with its edited flat config.
    printf "BEGIN; UPDATE notification_rules SET config='{\"features\":[]}' WHERE id='%s';\n" "$id" >&3
    deadline=$((SECONDS + 45))
    until [[ $(sql "SELECT count(*) FROM pg_stat_activity WHERE application_name='smoke_editor' AND state='idle in transaction' AND query LIKE 'UPDATE notification_rules%'") == 1 ]]; do
        (( SECONDS < deadline )) || fail "Concurrent Rule edit did not acquire the row"
        sleep 1
    done

    dc exec -T -e PGAPPNAME=smoke_backfill postgres psql -Xq -U postgres -d postgres -v ON_ERROR_STOP=1 \
        <"${backfills[4]}" >"$artifacts/concurrent-backfill.log" 2>&1 3>&- &
    migration=$!
    until [[ $(sql "SELECT count(*) FROM pg_stat_activity m JOIN pg_stat_activity e ON e.pid=ANY(pg_blocking_pids(m.pid))
        WHERE m.application_name='smoke_backfill' AND e.application_name='smoke_editor'") == 1 ]]; do
        (( SECONDS < deadline )) || fail "Rule backfill did not wait for the concurrent edit"
        sleep 1
    done
    echo 'COMMIT;' >&3
    exec 3>&-
    wait "$editor" && wait "$migration" || fail "Concurrent Rule edit or migration failed"
    [[ $(sql "SELECT config = '{\"features\":[]}'::jsonb FROM notification_rules WHERE id='$id'") == t ]] ||
        fail "Backfill lost the concurrent Rule edit"
}
