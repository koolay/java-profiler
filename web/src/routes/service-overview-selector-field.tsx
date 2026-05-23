import { ChevronDown, Loader2 } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState } from "react";
import type { KeyboardEvent } from "react";

type SelectorFieldProps = {
  label: string;
  value: string;
  candidates: string[];
  placeholder?: string;
  loading?: boolean;
  onChange: (value: string) => void;
};

function normalize(value: string) {
  return value.trim().toLowerCase();
}

function dedupe(values: string[]) {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))];
}

export function SelectorField({ label, value, candidates, placeholder, loading, onChange }: SelectorFieldProps) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const rootRef = useRef<HTMLLabelElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const listboxId = useId();

  const options = useMemo(() => {
    const current = value.trim();
    const normalized = normalize(current);
    const unique = dedupe(candidates);
    const ranked = normalized
      ? unique.filter((candidate) => normalize(candidate).includes(normalized))
      : unique;
    return current && !ranked.includes(current) ? [current, ...ranked] : ranked;
  }, [candidates, value]);

  useEffect(() => {
    if (!open) return;
    setActiveIndex((current) => {
      if (options.length === 0) return 0;
      return Math.min(current, options.length - 1);
    });
  }, [open, options.length]);

  useEffect(() => {
    function onPointerDown(event: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, []);

  const close = () => setOpen(false);

  const select = (nextValue: string) => {
    onChange(nextValue);
    setOpen(false);
    setActiveIndex(0);
    inputRef.current?.focus();
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((current) => (open ? Math.min(current + 1, Math.max(options.length - 1, 0)) : 0));
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((current) => (open ? Math.max(current - 1, 0) : Math.max(options.length - 1, 0)));
      return;
    }
    if (event.key === "Enter") {
      if (open && options[activeIndex]) {
        event.preventDefault();
        select(options[activeIndex]);
      }
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      close();
    }
  };

  return (
    <label className={`context-field selector-field${open ? " is-open" : ""}`} ref={rootRef}>
      <div className="selector-field-head">
        <span>{label}</span>
        {loading ? (
          <span className="selector-status" aria-label="Refreshing suggestions" title="Refreshing suggestions">
            <Loader2 size={12} className="selector-spinner" />
          </span>
        ) : null}
      </div>
      <div className="selector-combobox">
        <input
          aria-activedescendant={open && options[activeIndex] ? `${listboxId}-${activeIndex}` : undefined}
          aria-autocomplete="list"
          aria-controls={listboxId}
          aria-expanded={open}
          aria-haspopup="listbox"
          autoComplete="off"
          placeholder={placeholder}
          ref={inputRef}
          role="combobox"
          spellCheck={false}
          value={value}
          onChange={(event) => {
            onChange(event.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={handleKeyDown}
        />
        <button
          aria-label={`Show ${label.toLowerCase()} suggestions`}
          aria-expanded={open}
          className="selector-toggle"
          onClick={() => setOpen((current) => !current)}
          onMouseDown={(event) => event.preventDefault()}
          type="button"
        >
          <ChevronDown size={14} />
        </button>
      </div>
      {open ? (
        <div className="selector-dropdown" id={listboxId} role="listbox">
          {loading ? (
            <>
              <div className="selector-loading-row" aria-hidden="true" />
              <div className="selector-loading-row selector-loading-short" aria-hidden="true" />
              <div className="selector-loading-row selector-loading-medium" aria-hidden="true" />
            </>
          ) : options.length > 0 ? (
            options.map((option, index) => (
              <button
                aria-selected={index === activeIndex}
                className="selector-option"
                id={`${listboxId}-${index}`}
                key={option}
                onClick={() => select(option)}
                onMouseEnter={() => setActiveIndex(index)}
                onMouseDown={(event) => event.preventDefault()}
                role="option"
                type="button"
              >
                <span>{option}</span>
                {option === value.trim() ? <small>Current</small> : null}
              </button>
            ))
          ) : (
            <div className="selector-empty-state">
              No matching suggestions. Keep typing or press Enter to use the current value.
            </div>
          )}
        </div>
      ) : null}
    </label>
  );
}
