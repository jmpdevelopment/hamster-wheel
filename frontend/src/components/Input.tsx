interface InputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> {
  size?: "sm" | "md";
}

const sizeClasses = {
  sm: "px-2 py-1 text-xs",
  md: "px-2 py-1.5 text-sm",
} as const;

const baseClasses =
  "w-full rounded bg-hw-bg border border-hw-border text-hw-text placeholder-hw-text-muted focus:outline-none focus:border-hw-accent focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg transition-colors duration-150 disabled:opacity-50 disabled:cursor-not-allowed";

export function Input({ size = "md", className, ...rest }: InputProps) {
  return (
    <input
      className={`${baseClasses} ${sizeClasses[size]}${className ? ` ${className}` : ""}`}
      {...rest}
    />
  );
}
