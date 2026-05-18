const form = document.querySelector("#download-form");
const urlInput = document.querySelector("#url-input");
const formatSelect = document.querySelector("#format-select");
const qualitySelect = document.querySelector("#quality-select");
const qualityGroup = document.querySelector("#quality-group");
const optionsGroup = document.querySelector("#options-group");
const submitBtn = document.querySelector("#submit-btn");
const statusPanel = document.querySelector("#status-panel");
const statusText = document.querySelector("#status-text");

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
  return {
    url: urlInput.value.trim(),
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
  statusText.textContent = "Encolando job...";
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
    statusText.textContent = "Descargando...";
    return;
  }

  if (data.status === "completed") {
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
