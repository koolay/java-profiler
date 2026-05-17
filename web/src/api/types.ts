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
