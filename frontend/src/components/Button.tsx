import { Spinner } from "./Spinner";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "danger" | "ghost";
  size?: "sm" | "md";
  loading?: boolean;
}

const variantClasses = {
  primary: "bg-hw-accent text-hw-bg hover:bg-hw-accent-hover",
  secondary:
    "border border-hw-border text-hw-text-muted hover:text-hw-text hover:border-hw-text-muted",
  danger: "bg-hw-danger text-white hover:bg-hw-danger/80",
  ghost: "text-hw-text-muted hover:text-hw-text hover:bg-hw-surface-hover",
} as const;

const sizeClasses = {
  sm: "px-2 py-1 text-xs",
  md: "px-3 py-1.5 text-sm",
} as const;

const baseClasses =
  "inline-flex items-center justify-center gap-1.5 rounded font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg disabled:opacity-50 disabled:cursor-not-allowed";

export function Button({
  variant = "primary",
  size = "md",
  loading = false,
  disabled,
  children,
  className,
  ...rest
}: ButtonProps) {
  return (
    <button
      disabled={disabled || loading}
      className={`${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]}${className ? ` ${className}` : ""}`}
      {...rest}
    >
      {loading && <Spinner size="sm" />}
      {children}
    </button>
  );
}
