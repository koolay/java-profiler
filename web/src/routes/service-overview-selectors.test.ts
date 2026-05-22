import { expect, test } from "vitest";
import { buildSelectorCatalog } from "./service-overview-selectors";

test("builds distinct namespace/service/pod suggestions and keeps current values", () => {
  const catalog = buildSelectorCatalog(
    [
      { namespace: "prod", service: "checkout", pod: "checkout-1" },
      { namespace: "prod", service: "checkout", pod: "checkout-2" },
      { namespace: "prod", service: "payments", pod: "payments-1" },
      { namespace: "staging", service: "checkout", pod: "checkout-staging" },
      { namespace: "prod", service: "checkout", pod: "checkout-1" },
    ],
    { namespace: "prod", service: "checkout", pod: "checkout-current" },
  );

  expect(catalog.namespaces).toEqual(["prod", "staging"]);
  expect(catalog.services).toEqual(["checkout", "payments"]);
  expect(catalog.pods).toEqual(["checkout-current", "checkout-1", "checkout-2"]);
});

test("falls back to broader suggestions when the current selector has no matches", () => {
  const catalog = buildSelectorCatalog(
    [
      { namespace: "prod", service: "checkout", pod: "checkout-1" },
      { namespace: "prod", service: "payments", pod: "payments-1" },
    ],
    { namespace: "missing", service: "ghost", pod: "phantom" },
  );

  expect(catalog.namespaces).toEqual(["missing", "prod"]);
  expect(catalog.services).toEqual(["ghost", "checkout", "payments"]);
  expect(catalog.pods).toEqual(["phantom", "checkout-1", "payments-1"]);
});
