export interface ParsedLogLine {
  timestamp: string;
  severity: "INFO" | "ERROR";
  message: string;
}

export function parseLogLine(line: string): ParsedLogLine | null {
  const match = line.match(
    /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)\s+(INFO|ERROR)\s+(.*)$/,
  );
  if (!match) return null;
  return {
    timestamp: match[1],
    severity: match[2] as "INFO" | "ERROR",
    message: match[3],
  };
}
