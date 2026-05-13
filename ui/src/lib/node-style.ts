import type { Status } from "../types/state";

type Theme = "light" | "dark";

interface NodeStyle {
  fill: string;
  stroke: string;
  fontcolor: string;
}

const LIGHT: Record<string, NodeStyle> = {
  done:     { fill: "lightgreen",   stroke: "#22c55e", fontcolor: "black" },
  doing:    { fill: "lightblue",    stroke: "#3b82f6", fontcolor: "black" },
  error:    { fill: "mistyrose",    stroke: "#ef4444", fontcolor: "black" },
  undone:   { fill: "moccasin",     stroke: "#f97316", fontcolor: "black" },
  wait:     { fill: "lightyellow",  stroke: "#f59e0b", fontcolor: "black" },
  hold:     { fill: "lightgray",    stroke: "#a1a1aa", fontcolor: "black" },
  do:       { fill: "white",        stroke: "#a1a1aa", fontcolor: "black" },
  undo:     { fill: "white",        stroke: "#a1a1aa", fontcolor: "black" },
  undoing:  { fill: "white",        stroke: "#a1a1aa", fontcolor: "black" },
  abort:    { fill: "white",        stroke: "#a1a1aa", fontcolor: "black" },
  default:  { fill: "white",        stroke: "#a1a1aa", fontcolor: "black" },
};

const DARK: Record<string, NodeStyle> = {
  done:     { fill: "#4ade80", stroke: "#86efac", fontcolor: "black" },
  doing:    { fill: "#60a5fa", stroke: "#93c5fd", fontcolor: "black" },
  error:    { fill: "#f87171", stroke: "#fca5a5", fontcolor: "black" },
  undone:   { fill: "#fb923c", stroke: "#fdba74", fontcolor: "black" },
  wait:     { fill: "#fbbf24", stroke: "#fde68a", fontcolor: "black" },
  hold:     { fill: "#a1a1aa", stroke: "#52525b", fontcolor: "white" },
  do:       { fill: "#ffffff", stroke: "#52525b", fontcolor: "black" },
  undo:     { fill: "#52525b", stroke: "#3f3f46", fontcolor: "white" },
  undoing:  { fill: "#52525b", stroke: "#3f3f46", fontcolor: "white" },
  abort:    { fill: "#52525b", stroke: "#3f3f46", fontcolor: "white" },
  default:  { fill: "#52525b", stroke: "#3f3f46", fontcolor: "white" },
};

export function applyNodeStyle(nodeG: SVGGElement, status: Status, theme: Theme) {
  const map = theme === "light" ? LIGHT : DARK;
  const s = map[status] ?? map.default;

  for (const shape of nodeG.querySelectorAll<SVGElement>("ellipse, polygon")) {
    shape.setAttribute("fill", s.fill);
    shape.setAttribute("stroke", s.stroke);
    shape.setAttribute("stroke-width", "1.5");
  }

  for (const text of nodeG.querySelectorAll<SVGElement>("text")) {
    text.setAttribute("fill", s.fontcolor);
  }
}
