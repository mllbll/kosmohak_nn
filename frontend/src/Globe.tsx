import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/addons/controls/OrbitControls.js";
import { geoEquirectangular, geoPath, geoGraticule10 } from "d3-geo";
import { Plus, Minus, RotateCcw } from "lucide-react";
import { land, countries, mapNodes, nodeColor, orbitPoints } from "./geo";
import type { Layers, Scenario, Snapshot } from "./types";
interface Props {
  scenario: Scenario;
  snapshot?: Snapshot;
  layers: Layers;
  selected: string;
  onSelect: (id: string) => void;
  onFallback: () => void;
}
const vector = (p: [number, number, number]) =>
  new THREE.Vector3(p[0] / 6371, p[2] / 6371, -p[1] / 6371);
function disposeObject(object: THREE.Object3D) {
  object.traverse((o) => {
    const mesh = o as THREE.Mesh;
    mesh.geometry?.dispose();
    if (mesh.material) {
      for (const m of Array.isArray(mesh.material)
        ? mesh.material
        : [mesh.material]) {
        (m as THREE.MeshBasicMaterial).map?.dispose();
        m.dispose();
      }
    }
  });
}
export default function Globe(props: Props) {
  const host = useRef<HTMLDivElement>(null),
    api = useRef<{
      scene: THREE.Scene;
      camera: THREE.PerspectiveCamera;
      controls: OrbitControls;
      group: THREE.Group;
      labels: HTMLDivElement;
      labelsData: { el: HTMLButtonElement; position: THREE.Vector3 }[];
    } | null>(null),
    propsRef = useRef(props),
    [ready, setReady] = useState(0),
    [failed, setFailed] = useState(false);
  propsRef.current = props;
  useEffect(() => {
    const el = host.current;
    if (!el) return;
    let renderer: THREE.WebGLRenderer;
    try {
      renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    } catch {
      setFailed(true);
      return;
    }
    renderer.setPixelRatio(Math.min(devicePixelRatio, 2));
    renderer.outputColorSpace = THREE.SRGBColorSpace;
    el.appendChild(renderer.domElement);
    const scene = new THREE.Scene(),
      camera = new THREE.PerspectiveCamera(38, 1, 0.01, 50);
    camera.position.set(0.9, 2.35, -2.1);
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.09;
    controls.minDistance = 1.45;
    controls.maxDistance = 5;
    controls.enablePan = false;
    controls.rotateSpeed = 0.55;
    const canvas = document.createElement("canvas");
    canvas.width = 2048;
    canvas.height = 1024;
    const ctx = canvas.getContext("2d")!;
    const projection = geoEquirectangular()
        .scale(2048 / (2 * Math.PI))
        .translate([1024, 512]),
      path = geoPath(projection, ctx);
    ctx.fillStyle = "#071c31";
    ctx.fillRect(0, 0, 2048, 1024);
    ctx.beginPath();
    path(land);
    ctx.fillStyle = "#294c68";
    ctx.fill();
    ctx.strokeStyle = "#4d7690";
    ctx.lineWidth = 0.65;
    ctx.stroke();
    ctx.beginPath();
    path(countries);
    ctx.strokeStyle = "#385c77";
    ctx.lineWidth = 0.45;
    ctx.stroke();
    ctx.beginPath();
    path(geoGraticule10());
    ctx.strokeStyle = "#24465e";
    ctx.lineWidth = 0.55;
    ctx.stroke();
    const texture = new THREE.CanvasTexture(canvas);
    texture.colorSpace = THREE.SRGBColorSpace;
    texture.anisotropy = renderer.capabilities.getMaxAnisotropy();
    const earth = new THREE.Mesh(
      new THREE.SphereGeometry(1, 96, 64),
      new THREE.MeshPhongMaterial({
        map: texture,
        shininess: 9,
        specular: 0x183747,
      }),
    );
    scene.add(earth);
    scene.add(new THREE.AmbientLight(0xb8d8f2, 1.65));
    const sun = new THREE.DirectionalLight(0xc6e9ff, 2.5);
    sun.position.set(-3, 5, 2);
    scene.add(sun);
    const atmosphere = new THREE.Mesh(
      new THREE.SphereGeometry(1.016, 64, 48),
      new THREE.ShaderMaterial({
        transparent: true,
        side: THREE.BackSide,
        depthWrite: false,
        uniforms: {},
        vertexShader:
          "varying vec3 vNormal; varying vec3 vView; void main(){vec4 p=modelViewMatrix*vec4(position,1.0);vNormal=normalize(normalMatrix*normal);vView=normalize(-p.xyz);gl_Position=projectionMatrix*p;}",
        fragmentShader:
          "varying vec3 vNormal; varying vec3 vView; void main(){float a=pow(1.0-abs(dot(normalize(vNormal),normalize(vView))),3.0);gl_FragColor=vec4(0.12,0.57,1.0,a*0.7);}",
      }),
    );
    scene.add(atmosphere);
    const group = new THREE.Group();
    scene.add(group);
    const labels = document.createElement("div");
    labels.className = "globe-labels";
    el.appendChild(labels);
    const state = {
      scene,
      camera,
      controls,
      group,
      labels,
      labelsData: [] as { el: HTMLButtonElement; position: THREE.Vector3 }[],
    };
    api.current = state;
    const resize = new ResizeObserver(() => {
      const w = el.clientWidth,
        h = el.clientHeight;
      renderer.setSize(w, h);
      camera.aspect = w / Math.max(h, 1);
      camera.updateProjectionMatrix();
    });
    resize.observe(el);
    let frame = 0;
    function animate() {
      frame = requestAnimationFrame(animate);
      controls.update();
      // DOM labels and WebGL must use the same camera pose in this frame.
      camera.updateMatrixWorld();
      for (const label of state.labelsData) {
        const ray = label.position.clone().sub(camera.position),
          distance = ray.length();
        ray.normalize();
        const b = camera.position.dot(ray),
          disc = b * b - (camera.position.lengthSq() - 1);
        const hit = disc >= 0 ? -b - Math.sqrt(disc) : Infinity;
        const p = label.position.clone().project(camera);
        const visible =
          !(hit > 0 && hit < distance - 0.015) &&
          p.z < 1 &&
          p.z > -1 &&
          Math.abs(p.x) < 1.1 &&
          Math.abs(p.y) < 1.1;
        label.el.style.display = visible ? "" : "none";
        label.el.style.transform =
          "translate(-12px, -50%) translate(" +
          ((p.x + 1) * el!.clientWidth) / 2 +
          "px," +
          ((-p.y + 1) * el!.clientHeight) / 2 +
          "px)";
      }
      renderer.render(scene, camera);
    }
    animate();
    setReady((x) => x + 1);
    const lost = (event: Event) => {
      event.preventDefault();
      setFailed(true);
    };
    renderer.domElement.addEventListener("webglcontextlost", lost);
    return () => {
      cancelAnimationFrame(frame);
      resize.disconnect();
      controls.dispose();
      renderer.domElement.removeEventListener("webglcontextlost", lost);
      disposeObject(scene);
      renderer.dispose();
      renderer.domElement.remove();
      labels.remove();
      api.current = null;
    };
  }, []);
  useEffect(() => {
    const state = api.current;
    if (!state) return;
    disposeObject(state.group);
    state.group.clear();
    state.labels.replaceChildren();
    state.labelsData = [];
    const nodes = mapNodes(props.scenario, props.snapshot),
      byId = new Map(nodes.map((n) => [n.id, n]));
    const path = props.snapshot?.route.path ?? [],
      routeSet = new Set(path);
    function line(points: THREE.Vector3[], color: string, opacity: number) {
      const geometry = new THREE.BufferGeometry().setFromPoints(points);
      state!.group.add(
        new THREE.Line(
          geometry,
          new THREE.LineBasicMaterial({ color, transparent: true, opacity }),
        ),
      );
    }
    if (props.layers.orbits)
      for (const plane of props.scenario.design.planes)
        line(
          orbitPoints(
            props.scenario,
            props.snapshot?.snapshot.t_s ?? 0,
            plane.raan_deg,
          ).map(vector),
          "#4a789d",
          0.42,
        );
    if (props.layers.links)
      for (const [a, b] of props.snapshot?.snapshot.edges ?? []) {
        const na = byId.get(a),
          nb = byId.get(b);
        if (na && nb && na.status === "active" && nb.status === "active")
          line([vector(na.position), vector(nb.position)], "#608fab", 0.3);
      }
    if (props.layers.route)
      for (let i = 1; i < path.length; i++) {
        const a = byId.get(path[i - 1]),
          b = byId.get(path[i]);
        if (a && b) {
          const from = vector(a.position),
            to = vector(b.position),
            d = to.clone().sub(from);
          const mesh = new THREE.Mesh(
            new THREE.CylinderGeometry(0.003, 0.003, d.length(), 6),
            new THREE.MeshBasicMaterial({ color: "#32e4ff" }),
          );
          mesh.position.copy(from.clone().add(to).multiplyScalar(0.5));
          mesh.quaternion.setFromUnitVectors(
            new THREE.Vector3(0, 1, 0),
            d.normalize(),
          );
          state.group.add(mesh);
        }
      }
    for (const n of nodes) {
      if (
        (n.kind === "satellite" && !props.layers.satellites) ||
        (n.kind !== "satellite" && !props.layers.sites)
      )
        continue;
      const pos = vector(n.position);
      if (n.kind !== "satellite") pos.multiplyScalar(1.003);
      const material = new THREE.MeshBasicMaterial({ color: nodeColor(n) });
      const mesh = new THREE.Mesh(
        n.kind === "gateway"
          ? new THREE.BoxGeometry(0.018, 0.018, 0.018)
          : new THREE.SphereGeometry(
              n.kind === "satellite" ? 0.008 : 0.012,
              10,
              8,
            ),
        material,
      );
      mesh.position.copy(pos);
      state.group.add(mesh);
      const button = document.createElement("button");
      // A new snapshot must never expose a label at the overlay origin.
      button.style.display = "none";
      button.className =
        "map-label " +
        n.kind +
        " " +
        n.status +
        (props.selected === n.id ? " chosen" : "");
      button.title =
        n.id +
        " · " +
        (n.status === "failed"
          ? "Отказ"
          : n.status === "pending"
            ? "Не запущен"
            : n.kind === "satellite"
              ? "Спутник"
              : n.kind === "client"
                ? "Наземный пункт"
                : "Шлюз");
      button.setAttribute("aria-label", button.title);
      const symbol = document.createElement("span");
      symbol.className = "node-symbol";
      symbol.style.color = nodeColor(n);
      symbol.textContent =
        n.status === "failed"
          ? "×"
          : n.kind === "gateway"
            ? "■"
            : n.kind === "client"
              ? "▲"
              : "●";
      button.append(symbol);
      if (
        props.layers.labels ||
        routeSet.has(n.id) ||
        props.selected === n.id
      ) {
        const text = document.createElement("span");
        text.textContent = n.id;
        button.append(text);
      }
      button.addEventListener("click", () => propsRef.current.onSelect(n.id));
      state.labels.append(button);
      state.labelsData.push({ el: button, position: pos });
    }
  }, [props.scenario, props.snapshot, props.layers, props.selected, ready]);
  function zoom(scale: number) {
    const s = api.current;
    if (s) {
      s.camera.position.multiplyScalar(scale);
      s.controls.update();
    }
  }
  return (
    <div className="globe-host" ref={host} aria-label="Трёхмерная карта Земли">
      {failed && (
        <div className="map-fallback">
          <p>3D недоступно в этом браузере</p>
          <button onClick={props.onFallback}>Открыть 2D-карту</button>
        </div>
      )}
      <div className="map-controls">
        <button aria-label="Приблизить" onClick={() => zoom(0.85)}>
          <Plus size={18} />
        </button>
        <button aria-label="Отдалить" onClick={() => zoom(1.15)}>
          <Minus size={18} />
        </button>
        <button
          aria-label="Сбросить положение глобуса"
          onClick={() => {
            api.current?.camera.position.set(0.9, 2.35, -2.1);
            api.current?.controls.update();
          }}
        >
          <RotateCcw size={17} />
        </button>
      </div>
    </div>
  );
}
