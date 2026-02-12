interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  "aria-label": string;
}

const baseClasses =
  "p-1 rounded text-hw-text-muted hover:text-hw-text transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-hw-accent focus-visible:ring-offset-1 focus-visible:ring-offset-hw-bg disabled:opacity-50 disabled:cursor-not-allowed";

export function IconButton({
  children,
  className,
  ...rest
}: IconButtonProps) {
  return (
    <button
      className={`${baseClasses}${className ? ` ${className}` : ""}`}
      {...rest}
    >
      {children}
    </button>
  );
}
