# Open Work

## Add baseline comparison to profile analysis

The current UI answers “what is hot in this window?” A comparison view would also answer “what changed compared with a normal window?”

The feature needs two time windows, query and UI state for both windows, and careful handling of the seven-day retention limit. It should follow the accepted single-window CPU workflow rather than replace it.

When implemented, the comparison must remain self-owned and must not add Pyroscope, Parca, or Grafana as a required backend dependency.
