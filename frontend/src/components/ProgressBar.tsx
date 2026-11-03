type ProgressBarProps = {
  value: number;
};

export function ProgressBar({ value }: ProgressBarProps) {
  const safeValue = Math.max(0, Math.min(100, value));

  return (
    <div
      aria-label={`${safeValue.toFixed(1)}% complete`}
      aria-valuemax={100}
      aria-valuemin={0}
      aria-valuenow={safeValue}
      className="h-1.5 min-w-28 overflow-hidden rounded-full bg-slate-800"
      role="progressbar"
    >
      <div
        className="h-full rounded-full bg-sky-400 transition-[width] duration-300"
        style={{ width: `${safeValue}%` }}
      />
    </div>
  );
}
