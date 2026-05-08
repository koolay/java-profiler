CREATE TABLE IF NOT EXISTS java_profiler_profile_stacks
(
    cluster LowCardinality(String),
    namespace LowCardinality(String),
    service LowCardinality(String),
    pod String,
    container String,
    node LowCardinality(String),
    process_id UInt32,
    jvm_start_time DateTime64(9, 'UTC'),
    stack_id String,
    frames Array(String),
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = MergeTree
PARTITION BY toDate(created_at)
ORDER BY (cluster, namespace, service, pod, container, process_id, jvm_start_time, stack_id)
TTL expires_at DELETE;

CREATE TABLE IF NOT EXISTS java_profiler_profile_samples
(
    batch_id String,
    cluster LowCardinality(String),
    namespace LowCardinality(String),
    service LowCardinality(String),
    pod String,
    container String,
    node LowCardinality(String),
    process_id UInt32,
    jvm_start_time DateTime64(9, 'UTC'),
    profile_type LowCardinality(String),
    started_at DateTime64(9, 'UTC'),
    ended_at DateTime64(9, 'UTC'),
    stack_id String,
    sample_value UInt64,
    truncated UInt8 DEFAULT 0,
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = MergeTree
PARTITION BY toDate(started_at)
ORDER BY (cluster, namespace, service, profile_type, started_at, pod, process_id, stack_id)
TTL expires_at DELETE;

CREATE TABLE IF NOT EXISTS java_profiler_thread_snapshots
(
    batch_id String,
    cluster LowCardinality(String),
    namespace LowCardinality(String),
    service LowCardinality(String),
    pod String,
    container String,
    process_id UInt32,
    jvm_start_time DateTime64(9, 'UTC'),
    snapshot_at DateTime64(9, 'UTC'),
    thread_id Int64,
    native_thread_id String,
    thread_name String,
    daemon UInt8,
    thread_state LowCardinality(String),
    stack_frames Array(String),
    lock_owner String,
    blocked_lock String,
    waited_lock String,
    deadlock_cycle_id String,
    cpu_time_ns Nullable(UInt64),
    user_cpu_time_ns Nullable(UInt64),
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = MergeTree
PARTITION BY toDate(snapshot_at)
ORDER BY (cluster, namespace, service, pod, process_id, snapshot_at, thread_id)
TTL expires_at DELETE;

CREATE TABLE IF NOT EXISTS java_profiler_deadlock_events
(
    event_id String,
    cluster LowCardinality(String),
    namespace LowCardinality(String),
    service LowCardinality(String),
    pod String,
    process_id UInt32,
    jvm_start_time DateTime64(9, 'UTC'),
    event_at DateTime64(9, 'UTC'),
    cycle_id String,
    involved_threads Array(String),
    locks Array(String),
    blocking_frames Array(String),
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = MergeTree
PARTITION BY toDate(event_at)
ORDER BY (cluster, namespace, service, pod, process_id, event_at, cycle_id)
TTL expires_at DELETE;

CREATE TABLE IF NOT EXISTS java_profiler_target_status
(
    batch_id String,
    cluster LowCardinality(String),
    namespace LowCardinality(String),
    service LowCardinality(String),
    pod String,
    container String,
    process_id UInt32,
    jvm_start_time DateTime64(9, 'UTC'),
    status_at DateTime64(9, 'UTC'),
    desired_state LowCardinality(String),
    reason LowCardinality(String),
    message String,
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = MergeTree
PARTITION BY toDate(status_at)
ORDER BY (cluster, namespace, service, pod, container, process_id, status_at)
TTL expires_at DELETE;

CREATE TABLE IF NOT EXISTS java_profiler_ingestion_batches
(
    batch_id String,
    collector_id String,
    batch_type LowCardinality(String),
    received_at DateTime64(9, 'UTC'),
    status LowCardinality(String),
    retryable UInt8,
    payload_hash String,
    message String,
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 7 DAY
)
ENGINE = ReplacingMergeTree(received_at)
PARTITION BY toDate(received_at)
ORDER BY (batch_id, batch_type)
TTL expires_at DELETE;

CREATE TABLE IF NOT EXISTS java_profiler_artifact_index
(
    artifact_id String,
    batch_id String,
    artifact_type LowCardinality(String),
    path String,
    checksum String,
    created_at DateTime64(9, 'UTC') DEFAULT now64(9),
    expires_at DateTime DEFAULT toDateTime(created_at) + INTERVAL 24 HOUR
)
ENGINE = MergeTree
PARTITION BY toDate(created_at)
ORDER BY (artifact_id, batch_id)
TTL expires_at DELETE;
