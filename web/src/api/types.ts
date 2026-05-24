export type ProfileType =
  | "java_cpu_nanoseconds"
  | "java_allocation_bytes"
  | "java_allocation_objects"
  | "java_lock_contention_count"
  | "java_lock_delay_nanoseconds"
  | "java_wall_clock_nanoseconds"
  | "java_io_wait_nanoseconds";

export type FlamegraphNode = {
  name: string;
  value: number;
  display_value?: string;
  children?: FlamegraphNode[];
};

export type ProfileValueSemantics = {
  value_unit: string;
  display_unit: string;
  percent_basis: string;
  baseline_description: string;
  window_seconds?: number;
};

export type PartialMetadata = {
  partial: boolean;
  reasons?: string[];
  scanned_samples?: number;
  omitted_nodes?: number;
};

export type FlamegraphResponse = {
  root: FlamegraphNode;
  metadata: PartialMetadata;
  semantics?: ProfileValueSemantics;
};

export type ProfileSelector = {
  namespace: string;
  service: string;
  pod: string;
};

export type ServiceSelectors = {
  targets: ProfileSelector[];
};

export type ProfileTargetSummary = {
  namespace: string;
  service: string;
  pod: string;
  container: string;
  process_id: number;
  jvm_start_time: string;
  profile_type: ProfileType | string;
  total_value: number;
  display_value: string;
  sample_count: number;
  percent_of_total: string;
  semantics: ProfileValueSemantics;
};

export type ServiceProfileSummary = {
  targets: ProfileTargetSummary[];
  partial: boolean;
};

export type TopStackRow = {
  symbol: string;
  location: string;
  profile_type: ProfileType | string;
  self: number;
  total: number;
  self_display?: string;
  total_display?: string;
  self_percent: string;
  total_percent: string;
  semantics?: ProfileValueSemantics;
};

export type AllocationSummary = {
  schema_version: number;
  requested_scope: AllocationSummaryScope;
  effective_scope: AllocationSummaryScope;
  coverage: {
    has_data: boolean;
    empty_state?: string;
    profile_type: ProfileType | string;
    total_value: number;
    value_unit: string;
    scanned_samples: number;
    returned_paths: number;
    returned_self_frames: number;
    omitted_paths_lower_bound: number;
    omitted_nodes_lower_bound: number;
    partial: boolean;
    partial_reasons?: string[];
    newest_profile_end?: string;
  };
  top_paths: Array<{
    rank: number;
    leaf_frame: string;
    total_value: number;
    self_value: number;
    percent: number;
    category: string;
    sample_count: number;
    path: string[];
  }>;
  top_self_frames: Array<{
    rank: number;
    frame: string;
    self_value: number;
    percent: number;
    category: string;
  }>;
  insights: Array<{
    severity: string;
    category: string;
    message_code: string;
    evidence_frame: string;
    evidence_value: number;
  }>;
  limitations: Array<{ code: string; message_code: string }>;
  semantics?: ProfileValueSemantics;
};

export type AllocationSummaryScope = {
  namespace: string;
  service: string;
  pod: string;
  container: string;
  jvm: string;
  start: string;
  end: string;
};

export type TargetStatusReason =
  | "disabled_by_metadata"
  | "temporary_expired"
  | "invalid_duration"
  | "unsupported_jvm"
  | "profiler_conflict"
  | "attach_failed"
  | "upload_retryable"
  | "upload_dropped"
  | "storage_rejected"
  | "accepted"
  | string;

export type TargetStatus = {
  target?: { namespace?: string; service?: string; pod?: string; container?: string; process_id?: number };
  status_at?: string;
  desired_state: string;
  reason: TargetStatusReason;
  message: string;
};

export type DeadlockEvent = {
  event_id: string;
  cycle_id: string;
  involved_threads: string[];
  locks?: string[];
  blocking_frames?: string[];
};

export type ThreadDiagnosis = {
  busy_threads: Array<{ thread_id: number; thread_name: string; confidence: string; cpu_time_ns: number; stacks: string[] }>;
  slow_threads: Array<{ thread_id: number; thread_name: string; state: string; lock: string; stacks: string[] }>;
  partial: boolean;
};

export type IngestionHealth = {
  totals: {
    accepted: number;
    duplicate: number;
    retryable: number;
    rejected: number;
    dropped_samples: number;
    dropped_stacks: number;
    truncated_batches: number;
  };
  batches: Array<{
    batch_type: string;
    status: string;
    retryable: boolean;
    count: number;
    latest_at: string;
    last_message?: string;
  }>;
  partial: boolean;
};

export type JVMEvent = {
  event_id: string;
  batch_id?: string;
  target?: { namespace?: string; service?: string; pod?: string; container?: string; process_id?: number };
  event_type: string;
  event_at: string;
  duration_ns: number;
  collector?: string;
  action?: string;
  cause?: string;
  message?: string;
  stack_frames?: string[];
};

export type JVMEventEvidence = {
  events: JVMEvent[];
  partial: boolean;
};
