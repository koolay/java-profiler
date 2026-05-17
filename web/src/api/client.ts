import type { DeadlockEvent, FlamegraphResponse, IngestionHealth, JVMEventEvidence, TargetStatus, ThreadDiagnosis, TopStackRow } from "./types";

const apiBase = import.meta.env.VITE_API_BASE ?? "";

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { credentials: "include" });
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}`);
  }
  return response.json() as Promise<T>;
}

export function getFlamegraph(params: URLSearchParams) {
  return getJSON<FlamegraphResponse>(`/api/ui/v1/flamegraph?${params}`);
}

export function getTopStacks(params: URLSearchParams) {
  return getJSON<TopStackRow[]>(`/api/ui/v1/top-stacks?${params}`);
}

export function getThreadDiagnosis(params: URLSearchParams) {
  return getJSON<ThreadDiagnosis>(`/api/ui/v1/thread-diagnosis?${params}`);
}

export function getDeadlocks(params: URLSearchParams) {
  return getJSON<DeadlockEvent[]>(`/api/ui/v1/deadlocks?${params}`);
}

export function getTargetStatus(params: URLSearchParams) {
  return getJSON<TargetStatus[]>(`/api/ui/v1/target-status?${params}`);
}

export function getIngestionHealth() {
  return getJSON<IngestionHealth>("/api/ui/v1/ingestion");
}

export function getJVMEvents(params: URLSearchParams) {
  return getJSON<JVMEventEvidence>(`/api/ui/v1/jvm-events?${params}`);
}
