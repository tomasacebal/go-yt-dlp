const form = document.querySelector("#download-form");
const urlInput = document.querySelector("#url-input");
const formatSelect = document.querySelector("#format-select");
const qualitySelect = document.querySelector("#quality-select");
const qualityGroup = document.querySelector("#quality-group");
const optionsGroup = document.querySelector("#options-group");
const submitBtn = document.querySelector("#submit-btn");
const statusPanel = document.querySelector("#status-panel");
const progressFill = document.querySelector("#progress-fill");
const statusText = document.querySelector("#status-text");
const speedText = document.querySelector("#speed-text");
const etaText = document.querySelector("#eta-text");

let socket = null;

formatSelect.addEventListener("change", syncQualityVisibility);
syncQualityVisibility();

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  resetUIForStart();
  const payload = buildPayload();

  try {
    const response = await fetch("/api/download/start", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      const errorBody = await response.json().catch(() => ({}));
      throw new Error(errorBody.error || "No se pudo iniciar descarga");
    }

    const data = await response.json();
    openProgressSocket(data.jobId);
  } catch (error) {
    setError(error.message || "Error inesperado");
    resetFormState();
  }
});

function buildPayload() {
  const formatMode = formatSelect.value;
  const isAudioMode = formatMode === "audio";
  const normalizedURL = stripPlaylistListParam(urlInput.value);
  urlInput.value = normalizedURL;
  return {
    url: normalizedURL,
    flags: {
      format: "best",
      audioOnly: isAudioMode,
      quality: isAudioMode ? "best" : qualitySelect.value,
      embedSubs: false,
    },
  };
}

function resetUIForStart() {
  if (socket) {
    socket.close();
    socket = null;
  }
  submitBtn.disabled = true;
  statusPanel.classList.remove("hidden");
  optionsGroup.classList.add("hidden");
  progressFill.classList.remove("error");
  progressFill.style.width = "0%";
  statusText.textContent = "Encolando job...";
  speedText.textContent = "Velocidad: -";
  etaText.textContent = "ETA: -";
}

function openProgressSocket(jobId) {
  const proto = window.location.protocol === "https:" ? "wss" : "ws";
  socket = new WebSocket(`${proto}://${window.location.host}/ws/progress/${jobId}`);

  socket.onmessage = (message) => {
    const data = JSON.parse(message.data);
    applyProgressEvent(data);
  };

  socket.onerror = () => {
    setError("Conexion websocket fallida");
    resetFormState();
  };

  socket.onclose = () => {
    socket = null;
  };
}

function applyProgressEvent(data) {
  if (data.status === "queued") {
    statusText.textContent = "Job en cola...";
    return;
  }

  if (data.status === "downloading") {
    const progress = clamp(Number(data.progress || 0), 0, 100);
    if (progress > 0) {
      progressFill.style.width = `${progress}%`;
      statusText.textContent = `Descargando ${progress.toFixed(1)}%`;
    } else {
      statusText.textContent = "Descargando...";
    }
    speedText.textContent = `Velocidad: ${data.speed || "-"}`;
    etaText.textContent = `ETA: ${data.eta || "-"}`;
    return;
  }

  if (data.status === "completed") {
    progressFill.style.width = "100%";
    statusText.textContent = "Descarga completada. Iniciando descarga del archivo...";
    resetFormState();
    if (socket) {
      socket.close();
      socket = null;
    }
    window.location.href = `/api/download/file/${data.jobId}`;
    return;
  }

  if (data.status === "error") {
    setError(data.message || "Fallo la descarga");
    resetFormState();
  }
}

function setError(message) {
  progressFill.classList.add("error");
  statusText.textContent = message;
}

function resetFormState() {
  submitBtn.disabled = false;
  optionsGroup.classList.remove("hidden");
  syncQualityVisibility();
}

function syncQualityVisibility() {
  const isAudioMode = formatSelect.value === "audio";
  qualityGroup.classList.toggle("hidden", isAudioMode);
  qualitySelect.disabled = isAudioMode;
  if (isAudioMode) {
    qualitySelect.value = "best";
  }
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function stripPlaylistListParam(rawValue) {
  const trimmed = rawValue.trim();
  if (!trimmed) {
    return "";
  }

  try {
    const parsed = new URL(trimmed);
    if (!isYouTubeHost(parsed.hostname)) {
      return trimmed;
    }
    parsed.searchParams.delete("list");
    return parsed.toString();
  } catch {
    return trimmed;
  }
}

function isYouTubeHost(hostname) {
  const normalizedHost = hostname.toLowerCase();
  return (
    normalizedHost === "youtube.com" ||
    normalizedHost.endsWith(".youtube.com") ||
    normalizedHost === "youtu.be" ||
    normalizedHost.endsWith(".youtu.be") ||
    normalizedHost === "youtube-nocookie.com" ||
    normalizedHost.endsWith(".youtube-nocookie.com")
  );
}
