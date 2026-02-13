interface ShortcutsHelpProps {
  onClose: () => void;
}

const shortcuts = [
  { keys: ["j", "↓"], action: "Next job" },
  { keys: ["k", "↑"], action: "Previous job" },
  { keys: ["Enter"], action: "Open job detail" },
  { keys: ["Del", "⌫"], action: "Delete job (press twice)" },
  { keys: ["/", "Ctrl+F"], action: "Focus search" },
  { keys: ["Ctrl+R"], action: "Poll now" },
  { keys: [","], action: "Open settings" },
  { keys: ["?"], action: "Show shortcuts" },
  { keys: ["Esc"], action: "Close / cancel" },
];

export function ShortcutsHelp({ onClose }: ShortcutsHelpProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onClose}
      role="dialog"
      aria-label="Keyboard shortcuts"
    >
      <div
        className="bg-hw-surface border border-hw-border rounded-lg shadow-xl p-5 w-80"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-sm font-bold text-hw-text mb-3">
          Keyboard Shortcuts
        </h2>
        <dl className="space-y-1.5">
          {shortcuts.map((s) => (
            <div key={s.action} className="flex items-center justify-between">
              <dt className="flex gap-1">
                {s.keys.map((k) => (
                  <kbd
                    key={k}
                    className="px-1.5 py-0.5 text-xs font-mono rounded bg-hw-bg border border-hw-border text-hw-text-muted"
                  >
                    {k}
                  </kbd>
                ))}
              </dt>
              <dd className="text-sm text-hw-text-muted">{s.action}</dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  );
}
