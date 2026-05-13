import { useEffect, useEffectEvent, useRef } from "react";
import type { Change } from "../types/state";
import { generateDot } from "../lib/dot";
import { getViz } from "../lib/viz";
import { parseSvgLength } from "../lib/svg-units";
import { useTheme } from "../context/ThemeContext";

interface ChangeGraphProps {
  change: Change;
  selectedTaskId: string | null;
  onSelectTask: (id: string | null) => void;
}

interface ViewState {
  scale: number;
  x: number;
  y: number;
}

const MIN_SCALE = 0.1;
const ZOOM_SENSITIVITY = 0.001;

export default function ChangeGraph({
  change,
  selectedTaskId,
  onSelectTask,
}: ChangeGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const viewportRef = useRef<SVGGElement | null>(null);
  const prevSelectedRef = useRef<string | null>(null);
  const viewState = useRef<ViewState>({ scale: 1, x: 0, y: 0 });
  const initialViewState = useRef<ViewState>({ scale: 1, x: 0, y: 0 });
  const isDragging = useRef(false);
  const dragStart = useRef({ x: 0, y: 0 });
  const svgSize = useRef({ w: 0, h: 0 });
  const { theme } = useTheme();

  function applyTransform() {
    const g = viewportRef.current;
    if (!g) return;
    const { scale, x, y } = viewState.current;
    g.setAttribute("transform", `translate(${x},${y}) scale(${scale})`);
  }

  function screenToViewport(clientX: number, clientY: number) {
    const vp = viewportRef.current;
    const svg = svgRef.current;
    if (!vp || !svg) return { x: 0, y: 0 };
    const ctm = vp.getScreenCTM();
    if (!ctm) return { x: 0, y: 0 };
    const inv = ctm.inverse();
    const pt = svg.createSVGPoint();
    pt.x = clientX;
    pt.y = clientY;
    const mapped = pt.matrixTransform(inv);
    return { x: mapped.x, y: mapped.y };
  }

  useEffect(() => {
    let cancelled = false;

    async function render() {
      const viz = await getViz();
      if (cancelled) return;

      const dot = generateDot(change, theme);
      const svg = viz.renderSVGElement(dot, {
        graphAttributes: {
          nodesep: 0.4,
          ranksep: 0.8,
          pad: 0.3,
          bgcolor: "transparent",
          fontcolor: theme === "light" ? "black" : "white",
        },
        nodeAttributes: { fontcolor: theme === "light" ? "black" : "white" },
      });
      if (cancelled) return;

      const w = parseSvgLength(svg.getAttribute("width") ?? "");
      const h = parseSvgLength(svg.getAttribute("height") ?? "");
      svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
      svg.removeAttribute("width");
      svg.removeAttribute("height");
      svg.style.width = "100%";
      svg.style.height = "100%";
      svgSize.current = { w, h };

      const viewport = document.createElementNS("http://www.w3.org/2000/svg", "g");
      viewport.setAttribute("class", "graph-viewport");
      while (svg.firstChild) {
        viewport.appendChild(svg.firstChild);
      }
      svg.appendChild(viewport);

      const nodes = viewport.querySelectorAll<SVGGElement>(".node");
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

      svg.addEventListener("wheel", (e) => {
        e.preventDefault();
        const { scale, x, y } = viewState.current;
        const delta = -e.deltaY * ZOOM_SENSITIVITY;
        const factor = 1 + delta;
        const newScale = Math.max(MIN_SCALE, scale * factor);

        const pt = screenToViewport(e.clientX, e.clientY);

        viewState.current = {
          scale: newScale,
          x: x + pt.x * (scale - newScale),
          y: y + pt.y * (scale - newScale),
        };
        applyTransform();
      }, { passive: false });

      svg.addEventListener("mousedown", (e) => {
        if ((e.target as Element).closest(".node")) return;
        isDragging.current = true;
        const s = viewState.current;
        dragStart.current = { x: e.clientX, y: e.clientY };
        const prevX = s.x;
        const prevY = s.y;
        const ctm = svg.getScreenCTM();
        svg.style.cursor = "grabbing";

        const onMove = (ev: MouseEvent) => {
          if (!isDragging.current || !ctm) return;
          const dx = (ev.clientX - dragStart.current.x) / ctm.a;
          const dy = (ev.clientY - dragStart.current.y) / ctm.d;
          viewState.current.x = prevX + dx;
          viewState.current.y = prevY + dy;
          applyTransform();
        };

        const onUp = () => {
          isDragging.current = false;
          svg.style.cursor = "default";
          window.removeEventListener("mousemove", onMove);
          window.removeEventListener("mouseup", onUp);
        };

        window.addEventListener("mousemove", onMove);
        window.addEventListener("mouseup", onUp);
      });

      svg.addEventListener("dblclick", () => {
        const { scale, x, y } = initialViewState.current;
        viewState.current = { scale, x, y };
        applyTransform();
      });

      const container = containerRef.current;
      if (!container) return;

      container.replaceChildren(svg);
      svgRef.current = svg;
      viewportRef.current = viewport;

      const cW = container.clientWidth;
      const cH = container.clientHeight;
      const scaleCover = Math.max(cW / w, cH / h);
      const scaleMeet = Math.min(cW / w, cH / h);
      const initialScale = scaleCover / scaleMeet;
      const initX = w * (1 - initialScale) / 2;
      const initY = h * (1 - initialScale) / 2;

      initialViewState.current = { scale: initialScale, x: initX, y: initY };
      viewState.current = { scale: initialScale, x: initX, y: initY };
      applyTransform();
    }

    render().catch((err) => {
      if (!cancelled) {
        console.error("cannot render change graph:", err);
      }
    });

    return () => {
      cancelled = true;
      svgRef.current = null;
      viewportRef.current = null;
    };
  }, [change, onSelectTask, theme]);

  useEffect(() => {
    const prevId = prevSelectedRef.current;
    const viewport = viewportRef.current;

    if (prevId) {
      const prevNode = viewport?.querySelector<SVGGElement>(`#task-${prevId}`);
      prevNode?.classList.remove("graph-node--selected");
    }

    if (selectedTaskId) {
      const node = viewport?.querySelector<SVGGElement>(`#task-${selectedTaskId}`);
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
      className="w-full h-full"
    />
  );
}
