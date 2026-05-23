import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { SelectorField } from "./service-overview-selector-field";

test("shows a loading indicator while selector suggestions refresh", () => {
  render(<SelectorField candidates={[]} label="Namespace" loading onChange={vi.fn()} value="prod" />);

  expect(screen.getByLabelText("Refreshing suggestions")).toBeInTheDocument();
});

test("opens the suggestion list and lets the user choose a value", () => {
  const onChange = vi.fn();
  render(<SelectorField candidates={["prod", "staging"]} label="Namespace" onChange={onChange} value="" />);

  const input = screen.getByRole("combobox", { name: "Namespace" });
  fireEvent.focus(input);

  fireEvent.click(screen.getByRole("option", { name: "prod" }));

  expect(onChange).toHaveBeenCalledWith("prod");
});

test("keyboard navigation lands on the first suggestion from a closed field", () => {
  const onChange = vi.fn();
  render(<SelectorField candidates={["prod", "staging"]} label="Namespace" onChange={onChange} value="" />);

  const input = screen.getByRole("combobox", { name: "Namespace" });
  fireEvent.keyDown(input, { key: "ArrowDown" });
  fireEvent.keyDown(input, { key: "Enter" });

  expect(onChange).toHaveBeenCalledWith("prod");
});
