interface EmptyStateProps {
  title: string;
  description: string;
}

export function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
      <h3 className="text-lg font-semibold text-hw-text-muted leading-relaxed">{title}</h3>
      <p className="mt-1 text-sm text-hw-text-muted leading-relaxed">{description}</p>
    </div>
  );
}
