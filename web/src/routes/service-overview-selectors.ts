import type { ProfileTargetSummary } from "../api/types";

export type SelectorCatalog = {
  namespaces: string[];
  services: string[];
  pods: string[];
};

type SelectorValue = {
  namespace: string;
  service: string;
  pod: string;
};

type SelectorTarget = Pick<ProfileTargetSummary, "namespace" | "service" | "pod">;

function normalize(value: string | undefined | null) {
  return value?.trim() ?? "";
}

function listValues(values: Array<string | undefined | null>, currentValue = "") {
  const counts = new Map<string, number>();
  const selected = normalize(currentValue);
  for (const raw of [currentValue, ...values]) {
    const value = normalize(raw);
    if (!value) continue;
    counts.set(value, (counts.get(value) ?? 0) + 1);
  }
  return [...counts.entries()]
    .sort((left, right) => {
      if (selected && left[0] === selected && right[0] !== selected) return -1;
      if (selected && right[0] === selected && left[0] !== selected) return 1;
      return right[1] - left[1] || left[0].localeCompare(right[0]);
    })
    .map(([value]) => value);
}

function selectTargets(targets: SelectorTarget[], selection: SelectorValue) {
  const namespace = normalize(selection.namespace);
  const service = normalize(selection.service);
  const namespacePool = namespace ? targets.filter((target) => target.namespace === namespace) : targets;
  const servicePool = service ? namespacePool.filter((target) => target.service === service) : namespacePool;
  return {
    namespacePool: namespacePool.length > 0 ? namespacePool : targets,
    servicePool: servicePool.length > 0 ? servicePool : namespacePool.length > 0 ? namespacePool : targets,
  };
}

export function buildSelectorCatalog(targets: SelectorTarget[], selection: Partial<SelectorValue> = {}): SelectorCatalog {
  const namespace = normalize(selection.namespace);
  const service = normalize(selection.service);
  const pod = normalize(selection.pod);
  const scoped = selectTargets(targets, { namespace, service, pod });
  return {
    namespaces: listValues(targets.map((target) => target.namespace), namespace),
    services: listValues(scoped.namespacePool.map((target) => target.service), service),
    pods: listValues(scoped.servicePool.map((target) => target.pod), pod),
  };
}
