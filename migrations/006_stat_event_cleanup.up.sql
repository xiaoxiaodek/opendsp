-- Auto-create monthly partitions for stat_event
CREATE OR REPLACE FUNCTION ensure_stat_event_partition(target_month TIMESTAMPTZ)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    start_date TIMESTAMPTZ;
    end_date TIMESTAMPTZ;
BEGIN
    start_date := date_trunc('month', target_month);
    end_date := start_date + INTERVAL '1 month';
    partition_name := 'stat_event_' || to_char(start_date, 'YYYYMM');

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF stat_event FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );

    EXECUTE format(
        'CREATE INDEX IF NOT EXISTS %I ON %I (event_type, event_time)',
        'idx_' || partition_name || '_type_time', partition_name
    );
END;
$$ LANGUAGE plpgsql;

-- Ensure current and next 2 months exist
SELECT ensure_stat_event_partition(NOW());
SELECT ensure_stat_event_partition(NOW() + INTERVAL '1 month');
SELECT ensure_stat_event_partition(NOW() + INTERVAL '2 months');

-- Cleanup: drop partitions older than retention_days
CREATE OR REPLACE FUNCTION cleanup_stat_event_partitions(retention_days INTEGER DEFAULT 90)
RETURNS SETOF TEXT AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
          AND tablename ~ '^stat_event_\d{6}$'
          AND tablename < 'stat_event_' || to_char(NOW() - (retention_days || ' days')::INTERVAL, 'YYYYMM')
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I', rec.tablename);
        RETURN NEXT 'dropped: ' || rec.tablename;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Delete raw events after they've been aggregated (keep 2 days for safety)
CREATE OR REPLACE FUNCTION delete_aggregated_events(before_time TIMESTAMPTZ)
RETURNS BIGINT AS $$
DECLARE
    deleted BIGINT;
BEGIN
    DELETE FROM stat_event WHERE event_time < before_time;
    GET DIAGNOSTICS deleted = ROW_COUNT;
    RETURN deleted;
END;
$$ LANGUAGE plpgsql;
