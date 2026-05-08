export type ProfileType =
  | "java_cpu_nanoseconds"
  | "java_allocation_bytes"
  | "java_allocation_objects"
  | "java_lock_contention_count"
  | "java_lock_delay_nanoseconds";

export type FlamegraphNode = {
  name: string;
  value: number;
  children?: FlamegraphNode[];
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
};

export type TargetStatus = {
  target?: { namespace?: string; service?: string; pod?: string; process_id?: number };
  desired_state: string;
  reason: string;
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
