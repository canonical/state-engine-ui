import { useEffect, useEffectEvent, useRef } from "react";
import type { Change } from "../types/state";
import { generateDot } from "../lib/dot";
import { getViz } from "../lib/viz";

interface ChangeGraphProps {
  change: Change;
  selectedTaskId: string | null;
  onSelectTask: (id: string | null) => void;
}

export default function ChangeGraph({
  change,
  selectedTaskId,
  onSelectTask,
}: ChangeGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const prevSelectedRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function render() {
      const viz = await getViz();
      if (cancelled) return;

      const dot = generateDot(change);
      const svg = viz.renderSVGElement(dot, {
        graphAttributes: {
          nodesep: 0.4,
          ranksep: 0.8,
          pad: 0.3,
          bgcolor: "transparent",
          fontcolor: "white",
        },
        nodeAttributes: { fontcolor: "white" },
      });
      if (cancelled) return;

      const nodes = svg.querySelectorAll<SVGGElement>(".node");
      for (const node of nodes) {
        node.style.cursor = "pointer";
      }

      svg.addEventListener("click", (e) => {
        const target = e.target as Element;
        const nodeG = target.closest<SVGGElement>(".node");
        if (!nodeG) {
          onSelectTask(null);
          return;
        }
        const id = nodeG.id;
        if (id.startsWith("task-")) {
          onSelectTask(id.slice(5));
        }
      });

      const container = containerRef.current;
      if (!container) return;

      container.replaceChildren(svg);
      svgRef.current = svg;
    }

    render().catch((err) => {
      if (!cancelled) {
        console.error("cannot render change graph:", err);
      }
    });

    return () => {
      cancelled = true;
      svgRef.current = null;
    };
  }, [change, onSelectTask]);

  useEffect(() => {
    const prevId = prevSelectedRef.current;
    const svg = svgRef.current;

    if (prevId) {
      const prevNode = svg?.querySelector<SVGGElement>(`#task-${prevId}`);
      prevNode?.classList.remove("graph-node--selected");
    }

    if (selectedTaskId) {
      const node = svg?.querySelector<SVGGElement>(`#task-${selectedTaskId}`);
      node?.classList.add("graph-node--selected");
    }

    prevSelectedRef.current = selectedTaskId;
  }, [selectedTaskId]);

  const onKeyDown = useEffectEvent((e: KeyboardEvent) => {
    if (e.key === "Escape") {
      onSelectTask(null);
    }
  });

  useEffect(() => {
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  return (
    <div
      ref={containerRef}
      style={{ overflow: "auto", width: "100%", minHeight: "400px" }}
    />
  );
}
