import { forwardRef, useRef, useCallback } from "react";
import { Input } from "./Input";
import { IconButton } from "./IconButton";

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
}

export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(
  function SearchInput({ value, onChange }, forwardedRef) {
    const internalRef = useRef<HTMLInputElement>(null);

    // Merge forwarded ref and internal ref so both point to the same element.
    const setRef = useCallback(
      (element: HTMLInputElement | null) => {
        (internalRef as React.MutableRefObject<HTMLInputElement | null>).current =
          element;
        if (typeof forwardedRef === "function") {
          forwardedRef(element);
        } else if (forwardedRef) {
          (forwardedRef as React.MutableRefObject<HTMLInputElement | null>).current =
            element;
        }
      },
      [forwardedRef]
    );

    return (
      <div className="relative">
        <svg
          className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-hw-text-muted pointer-events-none"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
            clipRule="evenodd"
          />
        </svg>

        <Input
          ref={setRef}
          size="sm"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="Search jobs..."
          aria-label="Search jobs"
          className="pl-7 pr-7"
        />

        {value && (
          <div className="absolute right-1 top-1/2 -translate-y-1/2">
            <IconButton
              aria-label="Clear search"
              onClick={() => {
                onChange("");
                internalRef.current?.focus();
              }}
            >
              <svg
                className="w-3.5 h-3.5"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
              </svg>
            </IconButton>
          </div>
        )}
      </div>
    );
  }
);
