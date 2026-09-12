import { useRef, useState } from "react";
import {
  geoAzimuthalEquidistant,
  geoNaturalEarth1,
  geoPath,
  geoGraticule10,
} from "d3-geo";
import { Plus, Minus, RotateCcw } from "lucide-react";
import {
  land,
  countries,
  mapNodes,
  nodeColor,
  positionLonLat,
  orbitPoints,
} from "./geo";
import type { Scenario, Snapshot, Layers } from "./types";
interface Props {
  scenario: Scenario;
  snapshot?: Snapshot;
  layers: Layers;
  selected: string;
  onSelect: (id: string) => void;
}
export default function Map2D({
  scenario,
  snapshot,
  layers,
  selected,
  onSelect,
}: Props) {
  const [polar, setPolar] = useState(true),
    [zoom, setZoom] = useState(1),
    [offset, setOffset] = useState([0, 0]),
    drag = useRef<{ x: number; y: number; ox: number; oy: number } | null>(
      null,
    );
  const projection = polar
    ? geoAzimuthalEquidistant()
        .rotate([-60, -90, 0])
        .scale(235)
        .translate([500, 280])
        .clipAngle(115)
    : geoNaturalEarth1().fitExtent(
        [
          [25, 25],
          [975, 535],
        ],
        { type: "Sphere" },
      );
  const path = geoPath(projection),
    nodes = mapNodes(scenario, snapshot),
    byId = new Map(nodes.map((n) => [n.id, n])),
    route = snapshot?.route.path ?? [];
  const edge = (a: string, b: string) => {
    const x = byId.get(a),
      y = byId.get(b);
    return x && y
      ? path({
          type: "LineString",
          coordinates: [positionLonLat(x.position), positionLonLat(y.position)],
        })
      : null;
  };
  return (
    <div className="map2d">
      <svg
        viewBox="0 0 1000 560"
        aria-label="Двумерная карта сети"
        onWheel={(e) =>
          setZoom((z) =>
            Math.max(0.75, Math.min(5, z * (e.deltaY > 0 ? 0.9 : 1.1))),
          )
        }
        onPointerDown={(e) => {
          if ((e.target as Element).closest("[data-node]")) return;
          drag.current = {
            x: e.clientX,
            y: e.clientY,
            ox: offset[0],
            oy: offset[1],
          };
          e.currentTarget.setPointerCapture(e.pointerId);
        }}
        onPointerMove={(e) => {
          if (drag.current) {
            const scale = 1000 / e.currentTarget.getBoundingClientRect().width;
            setOffset([
              drag.current.ox + (e.clientX - drag.current.x) * scale,
              drag.current.oy + (e.clientY - drag.current.y) * scale,
            ]);
          }
        }}
        onPointerUp={() => {
          drag.current = null;
        }}
        onPointerCancel={() => {
          drag.current = null;
        }}
      >
        <defs>
          <radialGradient id="map-ocean">
            <stop stopColor="#122f46" />
            <stop offset="1" stopColor="#081624" />
          </radialGradient>
        </defs>
        <rect width="1000" height="560" fill="url(#map-ocean)" />
        <g
          transform={
            "translate(" +
            offset[0] +
            " " +
            offset[1] +
            ") translate(500 280) scale(" +
            zoom +
            ") translate(-500 -280)"
          }
        >
          <path
            d={path(land) || ""}
            fill="#1b3b53"
            stroke="#41677f"
            strokeWidth=".6"
          />
          <path
            d={path(countries) || ""}
            fill="none"
            stroke="#31536c"
            strokeWidth=".5"
          />
          <path
            d={path(geoGraticule10()) || ""}
            fill="none"
            stroke="#486c85"
            strokeOpacity=".35"
            strokeWidth=".5"
          />
          {layers.orbits &&
            scenario.design.planes.map((p) => (
              <path
                key={p.id}
                d={
                  path({
                    type: "LineString",
                    coordinates: orbitPoints(
                      scenario,
                      snapshot?.snapshot.t_s ?? 0,
                      p.raan_deg,
                    ).map(positionLonLat),
                  }) || ""
                }
                fill="none"
                stroke="#7c9cb8"
                strokeDasharray="4 5"
                strokeOpacity=".4"
                strokeWidth=".8"
              />
            ))}
          {layers.links &&
            snapshot?.snapshot.edges.map(([a, b], i) => (
              <path
                key={i}
                d={edge(a, b) || ""}
                fill="none"
                stroke="#7997af"
                strokeOpacity=".3"
                strokeWidth="1"
              />
            ))}
          {layers.route &&
            route
              .slice(1)
              .map((b, i) => (
                <path
                  key={i}
                  d={edge(route[i], b) || ""}
                  fill="none"
                  stroke="#36dffc"
                  strokeWidth="2.5"
                />
              ))}
          {nodes
            .filter((n) =>
              n.kind === "satellite" ? layers.satellites : layers.sites,
            )
            .map((n) => {
              const ll = positionLonLat(n.position),
                point = projection(ll);
              const visible = path({ type: "Point", coordinates: ll });
              if (!point || !visible) return null;
              return (
                <g
                  key={n.id}
                  data-node={n.id}
                  transform={"translate(" + point[0] + " " + point[1] + ")"}
                  role="button"
                  tabIndex={0}
                  aria-label={n.id}
                  onClick={() => onSelect(n.id)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSelect(n.id);
                    }
                  }}
                  className="svg-node"
                >
                  <title>
                    {n.id} ·{" "}
                    {n.status === "failed"
                      ? "Отказ"
                      : n.status === "pending"
                        ? "Не запущен"
                        : "Работает"}
                  </title>
                  <circle r="11" fill="transparent" />
                  {selected === n.id && (
                    <circle r="10" stroke="#fff" fill="none" strokeWidth="1" />
                  )}
                  {n.status === "failed" ? (
                    <path
                      d="M-4 -4L4 4M-4 4L4 -4"
                      stroke="#ff6b7e"
                      strokeWidth="2"
                    />
                  ) : n.kind === "gateway" ? (
                    <rect
                      x="-5"
                      y="-5"
                      width="10"
                      height="10"
                      fill={nodeColor(n)}
                    />
                  ) : n.kind === "client" ? (
                    <path d="M0 -6L6 5H-6Z" fill={nodeColor(n)} />
                  ) : (
                    <circle
                      r="4"
                      fill={nodeColor(n)}
                      stroke="#d0e4f7"
                      strokeWidth=".8"
                    />
                  )}
                  {(layers.labels ||
                    route.includes(n.id) ||
                    selected === n.id) && (
                    <text
                      x="9"
                      y="-7"
                      fill="#e1edf7"
                      fontSize="11"
                      paintOrder="stroke"
                      stroke="#08121f"
                      strokeWidth="3"
                    >
                      {n.id}
                    </text>
                  )}
                </g>
              );
            })}
        </g>
      </svg>
      <div className="projection-control">
        <button
          className="small"
          onClick={() => {
            setPolar(!polar);
            setOffset([0, 0]);
            setZoom(1);
          }}
        >
          {polar ? "Полярная проекция" : "Весь мир"} ⇄
        </button>
      </div>
      <div className="map-controls">
        <button
          aria-label="Приблизить карту"
          onClick={() => setZoom((z) => Math.min(5, z * 1.2))}
        >
          <Plus size={18} />
        </button>
        <button
          aria-label="Отдалить карту"
          onClick={() => setZoom((z) => Math.max(0.75, z / 1.2))}
        >
          <Minus size={18} />
        </button>
        <button
          aria-label="Сбросить положение карты"
          onClick={() => {
            setOffset([0, 0]);
            setZoom(1);
          }}
        >
          <RotateCcw size={17} />
        </button>
      </div>
    </div>
  );
}
