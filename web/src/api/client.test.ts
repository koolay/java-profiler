import { afterEach, expect, test, vi } from "vitest";
import { getAllocationSummary } from "./client";
import { APIError } from "./types";

afterEach(() => {
  vi.restoreAllMocks();
});

test("preserves structured API errors", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          code: "namespace_required",
          message: "Allocation summary requires a namespace.",
          field: "namespace",
          suggested_action: "Select a namespace before opening allocation Top Table evidence.",
        }),
        { status: 400, statusText: "Bad Request", headers: { "content-type": "application/json" } },
      ),
    ),
  );

  await expect(getAllocationSummary(new URLSearchParams("profile_type=java_allocation_bytes"))).rejects.toMatchObject({
    name: "APIError",
    status: 400,
    code: "namespace_required",
    field: "namespace",
    suggestedAction: "Select a namespace before opening allocation Top Table evidence.",
    message: "Allocation summary requires a namespace.",
  } satisfies Partial<APIError>);
});

test("falls back to status text for non-json errors", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("nope", { status: 503, statusText: "Service Unavailable" })));

  await expect(getAllocationSummary(new URLSearchParams("namespace=prod&profile_type=java_allocation_bytes"))).rejects.toThrow("503 Service Unavailable");
});
