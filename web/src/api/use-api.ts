import { useEffect, useState } from "react";

type State<T> = {
  data: T | null;
  error: string | null;
  loading: boolean;
};

export function useAPI<T>(loader: () => Promise<T>, deps: unknown[], fallback: T): State<T> {
  const [state, setState] = useState<State<T>>({ data: fallback, error: null, loading: true });
  useEffect(() => {
    let active = true;
    setState((current) => ({ ...current, loading: true, error: null }));
    loader()
      .then((data) => {
        if (active) setState({ data, error: null, loading: false });
      })
      .catch((error: Error) => {
        if (active) setState({ data: fallback, error: error.message, loading: false });
      });
    return () => {
      active = false;
    };
  }, deps);
  return state;
}
