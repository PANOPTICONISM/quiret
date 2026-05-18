// PDF annotation adapter. Stored annotations encode rects in the page's
// text-layer CSS-pixel space *at the scale they were saved* (the JSON shape
// is { page, scale, rects: [{x,y,w,h}, ...] }). At render time we multiply
// rect dimensions by currentScale / savedScale so highlights stay locked to
// the underlying text across window resizes.
//
// The public surface is the four `*PDFHighlight` / `*PDFSelection` /
// `*PDFAnnotation` functions plus `drawPDFHighlightsForPage`. They all take
// `pdfState` ({ textLayer, scale }) as the format-specific state container
// owned by the parent component.

import { HIGHLIGHT_COLORS } from "./constants.js";

const COLOR_MAP = Object.fromEntries(
  HIGHLIGHT_COLORS.map((c) => [c.name, c.rgba]),
);

function highlightColor(color) {
  return COLOR_MAP[color] || COLOR_MAP.yellow;
}

function drawRects(textLayerDiv, annotation, pos, scaleRatio = 1) {
  for (const rect of pos.rects) {
    const overlay = document.createElement("div");
    overlay.className = "pdf-highlight";
    overlay.dataset.annotationId = annotation.id;
    overlay.style.position = "absolute";
    overlay.style.left = `${rect.x * scaleRatio}px`;
    overlay.style.top = `${rect.y * scaleRatio}px`;
    overlay.style.width = `${rect.w * scaleRatio}px`;
    overlay.style.height = `${rect.h * scaleRatio}px`;
    overlay.style.background = highlightColor(annotation.color);
    overlay.style.pointerEvents = "none";
    overlay.style.mixBlendMode = "multiply";
    textLayerDiv.appendChild(overlay);
  }
}

export function drawPDFHighlightsForPage(
  textLayerDiv,
  pageNum,
  annotations,
  currentScale,
) {
  for (const a of annotations) {
    let pos;
    try {
      pos = JSON.parse(a.cfi);
    } catch {
      continue;
    }
    if (pos.page !== pageNum) continue;
    if (!Array.isArray(pos.rects)) continue;
    const ratio = pos.scale && currentScale ? currentScale / pos.scale : 1;
    drawRects(textLayerDiv, a, pos, ratio);
  }
}

export function applyPDFHighlight(annotation, pdfState, currentPage) {
  if (!pdfState.textLayer || !annotation?.cfi) return;
  let pos;
  try {
    pos = JSON.parse(annotation.cfi);
  } catch {
    return;
  }
  if (pos.page !== currentPage || !Array.isArray(pos.rects)) return;
  const ratio =
    pos.scale && pdfState.scale ? pdfState.scale / pos.scale : 1;
  drawRects(pdfState.textLayer, annotation, pos, ratio);
}

export function removePDFHighlight(annotation, pdfState) {
  if (!pdfState.textLayer) return;
  pdfState.textLayer
    .querySelectorAll(`[data-annotation-id="${annotation.id}"]`)
    .forEach((o) => o.remove());
}

export function goToPDFAnnotation(positionString, renderPage) {
  try {
    const pos = JSON.parse(positionString);
    if (pos.page) renderPage(pos.page);
  } catch {}
}

// Returns { text, position } for the current selection if it lives inside
// pdfState.textLayer, otherwise null. Position is a JSON string ready to be
// persisted as the annotation's `cfi` field.
export function capturePDFSelection(pdfState, currentPage) {
  if (!pdfState.textLayer) return null;
  const selection = document.getSelection();
  if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
    return null;
  }
  const range = selection.getRangeAt(0);
  if (!pdfState.textLayer.contains(range.commonAncestorContainer)) return null;
  const text = selection.toString().trim();
  if (!text) return null;
  const layerRect = pdfState.textLayer.getBoundingClientRect();
  const rects = Array.from(range.getClientRects())
    .map((r) => ({
      x: r.left - layerRect.left,
      y: r.top - layerRect.top,
      w: r.width,
      h: r.height,
    }))
    .filter((r) => r.w > 0 && r.h > 0);
  if (rects.length === 0) return null;
  return {
    text,
    position: JSON.stringify({
      page: currentPage,
      scale: pdfState.scale,
      rects,
    }),
  };
}
