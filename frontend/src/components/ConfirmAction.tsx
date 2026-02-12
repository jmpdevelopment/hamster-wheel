import { useState, useEffect, useRef, useCallback, cloneElement } from "react";
import { Button } from "./Button";

interface ConfirmActionProps {
  onConfirm: () => void;
  confirmLabel?: string;
  timeout?: number;
  children: React.ReactElement;
}

export function ConfirmAction({
  onConfirm,
  confirmLabel = "Confirm",
  timeout = 3000,
  children,
}: ConfirmActionProps) {
  const [confirming, setConfirming] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!confirming || timeout <= 0) return;
    const timer = setTimeout(() => setConfirming(false), timeout);
    return () => clearTimeout(timer);
  }, [confirming, timeout]);

  useEffect(() => {
    if (!confirming) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(e.target as Node)
      ) {
        setConfirming(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [confirming]);

  const handleTriggerClick = useCallback(() => {
    setConfirming(true);
  }, []);

  const handleConfirm = useCallback(() => {
    onConfirm();
    setConfirming(false);
  }, [onConfirm]);

  const handleCancel = useCallback(() => {
    setConfirming(false);
  }, []);

  if (confirming) {
    return (
      <div ref={wrapperRef} className="inline-flex gap-2">
        <Button variant="danger" size="sm" onClick={handleConfirm}>
          {confirmLabel}
        </Button>
        <Button variant="secondary" size="sm" onClick={handleCancel}>
          Cancel
        </Button>
      </div>
    );
  }

  return (
    <div ref={wrapperRef} className="inline-flex">
      {cloneElement(children, { onClick: handleTriggerClick })}
    </div>
  );
}
