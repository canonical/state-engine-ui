import { useEffect, useEffectEvent, useMemo, useRef } from "react";
import type { Change, Status } from "../types/state";
import { generateDot } from "../lib/dot";
import { applyNodeStyle } from "../lib/node-style";
import { getViz } from "../lib/viz";
import { parseSvgLength } from "../lib/svg-units";
import { useTheme } from "../context/ThemeContext";

interface ChangeGraphProps {
  change: Change;
  selectedTaskId: string | null;
  steppingTaskId: string | null;
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
  steppingTaskId,
  onSelectTask,
}: ChangeGraphProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const viewportRef = useRef<SVGGElement | null>(null);
  const viewState = useRef<ViewState>({ scale: 1, x: 0, y: 0 });
  const initialViewState = useRef<ViewState>({ scale: 1, x: 0, y: 0 });
  const isDragging = useRef(false);
  const dragStart = useRef({ x: 0, y: 0 });
  const svgSize = useRef({ w: 0, h: 0 });
  const isFirstRender = useRef(true);
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

  const structuralKey = useMemo(() => {
    return change.tasks
      .map(
        (t) =>
          `${t.id}\0${t.kind}\0${t.waitFor.slice().sort().join(",")}\0${t.lanes.slice().sort().join(",")}`
      )
      .sort()
      .join("|");
  }, [change]);

  const statusMap = useMemo(() => {
    const m = new Map<string, Status>();
    for (const t of change.tasks) m.set(t.id, t.status);
    return m;
  }, [change]);

  const handleNodeClick = useEffectEvent((e: MouseEvent) => {
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

  const handleWheel = useEffectEvent((e: WheelEvent) => {
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
  });

  const handleMouseDown = useEffectEvent((e: MouseEvent) => {
    if ((e.target as Element).closest(".node")) return;
    isDragging.current = true;
    const s = viewState.current;
    dragStart.current = { x: e.clientX, y: e.clientY };
    const prevX = s.x;
    const prevY = s.y;
    const svg = svgRef.current;
    if (!svg) return;
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
      if (svgRef.current) svgRef.current.style.cursor = "default";
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };

    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  });

  const handleDblClick = useEffectEvent(() => {
    const { scale, x, y } = initialViewState.current;
    viewState.current = { scale, x, y };
    applyTransform();
  });

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

      svg.addEventListener("click", handleNodeClick);
      svg.addEventListener("wheel", handleWheel, { passive: false });
      svg.addEventListener("mousedown", handleMouseDown);
      svg.addEventListener("dblclick", handleDblClick);

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

      if (isFirstRender.current) {
        isFirstRender.current = false;
        viewState.current = { scale: initialScale, x: initX, y: initY };
      }

      applyTransform();
    }

    render().catch((err) => {
      if (!cancelled) {
        console.error("cannot render change graph:", err);
      }
    });

    return () => {
      cancelled = true;
      svgRef.current?.removeEventListener("click", handleNodeClick);
      svgRef.current?.removeEventListener("wheel", handleWheel);
      svgRef.current?.removeEventListener("mousedown", handleMouseDown);
      svgRef.current?.removeEventListener("dblclick", handleDblClick);
      svgRef.current = null;
      viewportRef.current = null;
    };
  }, [structuralKey, theme]); // eslint-disable-line react-hooks/exhaustive-deps -- change used for generateDot but re-renders are gated by structuralKey

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    const nodes = viewport.querySelectorAll<SVGGElement>(".node");
    for (const node of nodes) {
      const taskId = node.id.startsWith("task-") ? node.id.slice(5) : null;
      if (taskId == null) continue;
      const status = statusMap.get(taskId) ?? "default";
      applyNodeStyle(node, status, theme);

      node.classList.toggle("graph-node--selected", taskId === selectedTaskId);
      node.classList.toggle("graph-node--stepping", taskId === steppingTaskId);
    }
  }, [statusMap, selectedTaskId, steppingTaskId, theme]);

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
