import { useEffect, useMemo, useRef, useState } from "react";
import {
  Check,
  FileImage,
  FolderOpen,
  Heart,
  ImagePlus,
  Info,
  Loader2,
  Move,
  RotateCcw,
  Save,
  SlidersHorizontal,
  Sparkles,
  X,
  ZoomIn,
} from "lucide-react";
import {
  CreateIcon,
  InspectImage,
  Reveal,
  SelectImage,
  SelectOutput,
} from "../wailsjs/go/main/App";
import { BrowserOpenURL, OnFileDrop, OnFileDropOff } from "../wailsjs/runtime/runtime";

const defaultRadius = 220;
const defaultZoom = 1;
const donationURL = "https://ko-fi.com/enelass";
const websiteURL = "https://www.photonsec.com.au";
const appVersion = "1.3.10";

function App() {
  const [image, setImage] = useState(null);
  const [outputPath, setOutputPath] = useState("");
  const [radius, setRadius] = useState(defaultRadius);
  const [zoom, setZoom] = useState(defaultZoom);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [transparentBg, setTransparentBg] = useState(false);
  const [keepIntermediates, setKeepIntermediates] = useState(false);
  const [status, setStatus] = useState("idle");
  const [message, setMessage] = useState("");
  const [result, setResult] = useState(null);
  const [isPanning, setIsPanning] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const dragRef = useRef(null);
  const aboutButtonRef = useRef(null);
  const aboutCloseRef = useRef(null);

  useEffect(() => {
    if (!window.runtime?.OnFileDrop) {
      return undefined;
    }

    OnFileDrop((_, __, paths) => {
      if (paths?.length) {
        loadImage(paths[0]);
      }
    }, true);

    return () => OnFileDropOff();
  }, []);

  useEffect(() => {
    if (!aboutOpen) return undefined;

    aboutCloseRef.current?.focus();
    function closeOnEscape(event) {
      if (event.key === "Escape") {
        setAboutOpen(false);
      }
    }
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      window.removeEventListener("keydown", closeOnEscape);
      aboutButtonRef.current?.focus();
    };
  }, [aboutOpen]);

  const canCreate = Boolean(image?.path && outputPath && status !== "running");
  const sizeLabel = useMemo(() => formatBytes(image?.sizeBytes || 0), [image]);
  const previewRadius = useMemo(() => `${(radius / 1024) * 100}%`, [radius]);
  const roundness = useMemo(() => Math.round((radius / 512) * 100), [radius]);
  const zoomPercent = useMemo(() => Math.round(zoom * 100), [zoom]);
  const minimumZoom = useMemo(() => {
    if (!image?.width || !image?.height) return defaultZoom;
    return Math.min(image.width, image.height) / Math.max(image.width, image.height);
  }, [image]);
  const minimumZoomPercent = useMemo(() => Math.ceil(minimumZoom * 1000) / 10, [minimumZoom]);
  const previewGeometry = useMemo(() => {
    if (!image?.width || !image?.height) {
      return { width: 100, height: 100, left: 0, top: 0, overflowX: 0, overflowY: 0 };
    }
    const effectiveZoom = clamp(zoom, minimumZoom, 3);
    const shortSide = Math.min(image.width, image.height);
    const width = (image.width / shortSide) * effectiveZoom * 100;
    const height = (image.height / shortSide) * effectiveZoom * 100;
    const overflowX = Math.max(width - 100, 0);
    const overflowY = Math.max(height - 100, 0);
    return {
      width,
      height,
      left: (100 - width) / 2 + (pan.x / 100) * (overflowX / 2),
      top: (100 - height) / 2 + (pan.y / 100) * (overflowY / 2),
      overflowX,
      overflowY,
    };
  }, [image, minimumZoom, pan.x, pan.y, zoom]);
  const previewStyle = useMemo(
    () => ({
      "--preview-radius": previewRadius,
      "--preview-image-width": `${previewGeometry.width}%`,
      "--preview-image-height": `${previewGeometry.height}%`,
      "--preview-image-left": `${previewGeometry.left}%`,
      "--preview-image-top": `${previewGeometry.top}%`,
    }),
    [previewGeometry, previewRadius],
  );
  const canPan = Boolean(image && (previewGeometry.overflowX > 0.01 || previewGeometry.overflowY > 0.01));

  async function loadImage(path) {
    setStatus("loading");
    setMessage("");
    setResult(null);
    try {
      const info = await InspectImage(path);
      setImage(info);
      setOutputPath(info.defaultOutputPath);
      setZoom(defaultZoom);
      setPan({ x: 0, y: 0 });
      setStatus("idle");
    } catch (error) {
      setStatus("error");
      setMessage(errorMessage(error));
    }
  }

  async function browseImage() {
    setStatus("loading");
    setMessage("");
    setResult(null);
    try {
      const info = await SelectImage();
      if (info?.path) {
        setImage(info);
        setOutputPath(info.defaultOutputPath);
        setZoom(defaultZoom);
        setPan({ x: 0, y: 0 });
      }
      setStatus("idle");
    } catch (error) {
      setStatus("error");
      setMessage(errorMessage(error));
    }
  }

  async function chooseOutput() {
    setMessage("");
    try {
      const selected = await SelectOutput(outputPath || image?.defaultOutputPath || "");
      if (selected) {
        setOutputPath(selected);
      }
    } catch (error) {
      setStatus("error");
      setMessage(errorMessage(error));
    }
  }

  async function createIcon() {
    if (!canCreate) return;
    setStatus("running");
    setMessage("");
    setResult(null);

    try {
      const created = await CreateIcon({
        inputPath: image.path,
        outputPath,
        radius,
        zoom,
        panX: pan.x,
        panY: pan.y,
        transparentBg,
        keepIntermediates,
      });
      setResult(created);
      setStatus("success");
      setMessage(created.statusMessage);
    } catch (error) {
      setStatus("error");
      setMessage(errorMessage(error));
    }
  }

  async function revealResult() {
    if (result?.icnsPath) {
      await Reveal(result.icnsPath);
    }
  }

  function reset() {
    setImage(null);
    setOutputPath("");
    setRadius(defaultRadius);
    setZoom(defaultZoom);
    setPan({ x: 0, y: 0 });
    setTransparentBg(false);
    setKeepIntermediates(false);
    setStatus("idle");
    setMessage("");
    setResult(null);
  }

  function donate() {
    BrowserOpenURL(donationURL);
  }

  function beginPan(event) {
    if (!canPan) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture?.(event.pointerId);
    dragRef.current = {
      startX: event.clientX,
      startY: event.clientY,
      panX: pan.x,
      panY: pan.y,
      width: event.currentTarget.clientWidth,
      height: event.currentTarget.clientHeight,
      overflowX: previewGeometry.overflowX,
      overflowY: previewGeometry.overflowY,
    };
    setIsPanning(true);
  }

  function movePan(event) {
    const drag = dragRef.current;
    if (!drag) return;

    const maxX = (drag.width * drag.overflowX) / 200;
    const maxY = (drag.height * drag.overflowY) / 200;
    setPan({
      x: maxX > 0 ? clamp(drag.panX + ((event.clientX - drag.startX) / maxX) * 100, -100, 100) : 0,
      y: maxY > 0 ? clamp(drag.panY + ((event.clientY - drag.startY) / maxY) * 100, -100, 100) : 0,
    });
  }

  function endPan(event) {
    if (!dragRef.current) return;
    event.currentTarget.releasePointerCapture?.(event.pointerId);
    dragRef.current = null;
    setIsPanning(false);
  }

  return (
    <main className="app-shell">
      <header className="titlebar">
        <div className="brand">
          <div className="brand-mark">
            <Sparkles size={18} aria-hidden="true" />
          </div>
          <div>
            <h1>Icon Creator</h1>
            <p>macOS .icns studio</p>
          </div>
        </div>
        <div className="title-actions">
          <button className="donate-button" title="Donate to support this work" type="button" onClick={donate}>
            <Heart size={17} aria-hidden="true" />
            <span>Donate to support this work</span>
          </button>
          <button
            ref={aboutButtonRef}
            aria-haspopup="dialog"
            className="icon-button"
            title="About Icon Creator"
            type="button"
            onClick={() => setAboutOpen(true)}
          >
            <Info size={18} aria-hidden="true" />
          </button>
          <button className="icon-button" title="Reset" type="button" onClick={reset}>
            <RotateCcw size={18} aria-hidden="true" />
          </button>
        </div>
      </header>

      <section className="workspace">
        <div className="preview-pane">
          <div className="drop-target" style={{ "--wails-drop-target": "drop" }}>
            {image ? (
              <div className="preview-stage" style={previewStyle}>
                <div className="preview-card">
                  <div
                    aria-label={image.name}
                    className={`icon-preview ${canPan ? "can-pan" : ""} ${isPanning ? "is-panning" : ""}`}
                    role="img"
                    onPointerCancel={endPan}
                    onPointerDown={beginPan}
                    onPointerMove={movePan}
                    onPointerUp={endPan}
                  >
                    <img alt="" className="preview-image" draggable="false" src={image.previewDataURL} />
                  </div>
                  <div className="radius-outline" aria-hidden="true" />
                  <div className={`preview-tools ${canPan ? "" : "disabled"}`}>
                    <div className="move-pill">
                      <Move size={15} aria-hidden="true" />
                      <span>Move icon</span>
                    </div>
                    <button className="preview-center-button" type="button" onClick={() => setPan({ x: 0, y: 0 })}>
                      Center
                    </button>
                  </div>
                  <div className="preview-badges">
                    <div className="preview-badge">Shape {roundness}%</div>
                    <div className="preview-badge">Zoom {zoomPercent}%</div>
                    {transparentBg && <div className="preview-badge">Alpha</div>}
                  </div>
                </div>
              </div>
            ) : (
              <div className="empty-drop">
                <ImagePlus size={44} aria-hidden="true" />
                <div>
                  <h2>Drop image</h2>
                  <p>PNG, JPG, GIF, WebP, SVG, ICO, ICNS</p>
                </div>
              </div>
            )}
          </div>

          <div className="image-strip">
            <div className="meta-block">
              <FileImage size={18} aria-hidden="true" />
              <div>
                <strong>{image?.name || "No source selected"}</strong>
                <span>
                  {image
                    ? `${image.width} x ${image.height} px · ${sizeLabel}`
                    : "Ready for a source image"}
                </span>
              </div>
            </div>
            <button className="secondary-button" type="button" onClick={browseImage}>
              <FolderOpen size={17} aria-hidden="true" />
              Browse
            </button>
          </div>
        </div>

        <aside className="control-pane">
          <section className="control-group">
            <div className="section-title">
              <SlidersHorizontal size={18} aria-hidden="true" />
              <h2>Shape</h2>
            </div>
            <label className="range-row">
              <span>Shape feel</span>
              <strong>{roundness}%</strong>
            </label>
            <input
              className="radius-slider"
              max="100"
              min="0"
              step="1"
              type="range"
              value={roundness}
              onChange={(event) => setRadius(Math.round((Number(event.target.value) / 100) * 512))}
            />
            <div className="shape-endpoints" aria-hidden="true">
              <span>Rect</span>
              <span>Circle</span>
            </div>
            <label className="range-row">
              <span>Zoom</span>
              <strong>{zoomPercent}%</strong>
            </label>
            <input
              className="radius-slider"
              max="300"
              min={minimumZoomPercent}
              step="0.1"
              type="range"
              value={zoom * 100}
              onChange={(event) => setZoom(Number(event.target.value) / 100)}
            />
            <div className="shape-endpoints" aria-hidden="true">
              <span>Fit</span>
              <span>Tight</span>
            </div>
          </section>

          <section className="control-group">
            <div className="section-title">
              <Save size={18} aria-hidden="true" />
              <h2>Output</h2>
            </div>
            <label className="field-label" htmlFor="output-path">
              Base file
            </label>
            <div className="path-row">
              <input
                id="output-path"
                spellCheck="false"
                type="text"
                value={outputPath}
                onChange={(event) => setOutputPath(event.target.value)}
              />
              <button className="icon-button" title="Choose output file" type="button" onClick={chooseOutput}>
                <FolderOpen size={18} aria-hidden="true" />
              </button>
            </div>
            <p className="output-hint">Exports matching macOS .icns, Windows .ico, and PNG files.</p>
            <label
              className="toggle-row"
              title="Turns a solid connected outer color, such as white or off-white, into transparency."
            >
              <input
                checked={transparentBg}
                type="checkbox"
                onChange={(event) => setTransparentBg(event.target.checked)}
              />
              <span>Transparent outer color</span>
            </label>
            <label className="toggle-row">
              <input
                checked={keepIntermediates}
                type="checkbox"
                onChange={(event) => setKeepIntermediates(event.target.checked)}
              />
              <span>Keep working files</span>
            </label>
          </section>

          <button className="primary-button" disabled={!canCreate} type="button" onClick={createIcon}>
            {status === "running" ? (
              <Loader2 className="spin" size={19} aria-hidden="true" />
            ) : (
              <ZoomIn size={19} aria-hidden="true" />
            )}
            Create icons
          </button>

          <div className={`status-line ${status}`}>
            {status === "success" && <Check size={18} aria-hidden="true" />}
            <span>{message || "No extra files are kept by default."}</span>
          </div>

          {result && (
            <section className="result-panel">
              <div className="result-files">
                <div>
                  <strong>{result.icnsFileName}</strong>
                  <span>macOS · {formatBytes(result.icnsSize)}</span>
                </div>
                <div>
                  <strong>{result.icoFileName}</strong>
                  <span>Windows · {formatBytes(result.icoSize)}</span>
                </div>
                <div>
                  <strong>{result.pngFileName}</strong>
                  <span>PNG · {formatBytes(result.pngSize)}</span>
                </div>
              </div>
              <button className="secondary-button" type="button" onClick={revealResult}>
                <FolderOpen size={17} aria-hidden="true" />
                Reveal
              </button>
            </section>
          )}
        </aside>
      </section>

      {aboutOpen && (
        <div
          className="modal-backdrop"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) setAboutOpen(false);
          }}
        >
          <section aria-labelledby="about-title" aria-modal="true" className="about-dialog" role="dialog">
            <button
              ref={aboutCloseRef}
              aria-label="Close About"
              className="icon-button about-close"
              type="button"
              onClick={() => setAboutOpen(false)}
            >
              <X size={18} aria-hidden="true" />
            </button>
            <div className="brand-mark about-mark">
              <Sparkles size={24} aria-hidden="true" />
            </div>
            <div className="about-copy">
              <h2 id="about-title">Icon Creator</h2>
              <p className="about-version">Version {appVersion}</p>
              <p>Created by Florian Bidabe<br />Photon Security</p>
            </div>
            <div className="about-actions">
              <button className="secondary-button" type="button" onClick={() => BrowserOpenURL(websiteURL)}>
                photonsec.com.au
              </button>
              <button className="donate-button" type="button" onClick={donate}>
                <Heart size={17} aria-hidden="true" />
                Donate
              </button>
            </div>
          </section>
        </div>
      )}
    </main>
  );
}

function formatBytes(bytes) {
  if (!bytes) return "0 KB";
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function errorMessage(error) {
  if (!error) return "Something went wrong.";
  if (typeof error === "string") return error;
  return error.message || JSON.stringify(error);
}

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

export default App;
