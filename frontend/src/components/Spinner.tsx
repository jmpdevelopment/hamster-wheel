interface SpinnerProps {
  size?: "sm" | "md";
}

const sizeMap = {
  sm: 14,
  md: 18,
} as const;

export function Spinner({ size = "md" }: SpinnerProps) {
  const px = sizeMap[size];

  return (
    <svg
      className="animate-spin"
      width={px}
      height={px}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
    >
      <circle
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="3"
        strokeLinecap="round"
        opacity="0.25"
      />
      <path
        d="M12 2a10 10 0 0 1 10 10"
        stroke="currentColor"
        strokeWidth="3"
        strokeLinecap="round"
      />
    </svg>
  );
}
