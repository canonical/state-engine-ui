import { useState } from "react";
import { parseLogLine } from "../lib/parseLog";

interface TaskLogViewerProps {
  logs: string[];
  defaultExpanded?: boolean;
}

export default function TaskLogViewer({ logs, defaultExpanded = false }: TaskLogViewerProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  if (logs.length === 0) {
    return null;
  }

  return (
    <div className="mt-3">
      <button
        type="button"
        onClick={() => setIsExpanded((prev) => !prev)}
        className="text-xs text-gray-500 hover:text-gray-700 transition-colors"
      >
        {isExpanded ? "Hide logs" : `Show logs (${logs.length})`}
      </button>

      {isExpanded ? (
        <div className="mt-2 max-h-48 overflow-y-auto rounded bg-gray-50 p-3">
          {logs.map((line, index) => {
            const parsed = parseLogLine(line);
            if (!parsed) {
              return (
                <div key={index} className="font-mono text-xs text-gray-500">
                  {line}
                </div>
              );
            }

            return (
              <div
                key={index}
                className="font-mono text-xs leading-relaxed mb-1 last:mb-0"
                title={parsed.timestamp}
              >
                <span
                  className={
                    parsed.severity === "ERROR"
                      ? "text-red-600 font-medium"
                      : "text-gray-700"
                  }
                >
                  {parsed.severity === "ERROR" ? "⚠ " : "ℹ "}
                  {parsed.message}
                </span>
              </div>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
