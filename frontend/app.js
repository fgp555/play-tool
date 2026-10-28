const projectPathInput = document.querySelector("#project-path");
const keyPathInput = document.querySelector("#key-path");
const output = document.querySelector("#output");
const runStatus = document.querySelector("#run-status");
const projectError = document.querySelector("#project-error");
const buttons = [...document.querySelectorAll("button[data-action]")];
let running = false;

function setStatus(label, state) {
  runStatus.className = `run-status ${state}`;
  runStatus.lastChild.textContent = label;
}

function setBusy(value) {
  running = value;
  buttons.forEach((button) => { button.disabled = value; });
  document.querySelector("#pick-folder").disabled = value;
  document.querySelector("#pick-key").disabled = value;
  document.querySelector("#refresh-project").disabled = value;
}

async function refreshProject() {
  projectError.classList.add("hidden");
  const query = new URLSearchParams();
  if (projectPathInput.value.trim()) query.set("path", projectPathInput.value.trim());
  try {
    const response = await fetch(`/api/project?${query.toString()}`);
    const project = await response.json();
    if (project.path) projectPathInput.value = project.path;
    if (project.error) {
      document.querySelector("#project-name").textContent = "No se detectó el proyecto";
      projectError.textContent = project.error;
      projectError.classList.remove("hidden");
      document.querySelector("#package-name").textContent = "—";
      document.querySelector("#track-name").textContent = "—";
      document.querySelector("#gradle-root").textContent = "—";
      document.querySelector("#config-name").textContent = "—";
      document.querySelector("#credential-status").textContent = "Proyecto no detectado";
      return;
    }
    document.querySelector("#project-name").textContent = project.package || "Proyecto Android";
    document.querySelector("#package-name").textContent = project.package || "—";
    document.querySelector("#track-name").textContent = project.track || "internal";
    document.querySelector("#gradle-root").textContent = project.gradleRoot || "—";
    document.querySelector("#gradle-root").title = project.gradleRoot || "";
    document.querySelector("#config-name").textContent = project.configuration || "automática";
    document.querySelector("#credential-status").textContent = keyPathInput.value.trim()
      ? "JSON seleccionado para esta ventana"
      : project.keyConfigured
        ? "JSON de servicio detectado y disponible"
        : "Sin clave detectada: selecciona el JSON o configura PLAY_TOOL_KEY";
  } catch (error) {
    projectError.textContent = `No se pudo consultar el proyecto: ${error.message}`;
    projectError.classList.remove("hidden");
  }
}

async function choosePath(endpoint, input) {
  try {
    const response = await fetch(endpoint);
    const result = await response.json();
    if (!response.ok) throw new Error(result.error || "No se pudo abrir el selector.");
    if (result.path) {
      input.value = result.path;
      if (input === projectPathInput) await refreshProject();
      else document.querySelector("#credential-status").textContent = "JSON seleccionado para esta ejecución";
    }
  } catch (error) {
    output.textContent = error.message;
    setStatus("Error", "error");
  }
}

async function runAction(action) {
  if (running) return;
  if (action === "deploy" && !window.confirm("Deploy incrementará versionCode, compilará y publicará el AAB en Google Play. ¿Continuar?")) return;

  setBusy(true);
  setStatus("Ejecutando", "running");
  output.textContent = "";
  try {
    const response = await fetch("/api/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        action,
        path: projectPathInput.value.trim(),
        keyPath: keyPathInput.value.trim(),
      }),
    });
    if (!response.ok) {
      output.textContent = await response.text();
      setStatus("Error", "error");
      return;
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      output.textContent += decoder.decode(value, { stream: true });
      output.scrollTop = output.scrollHeight;
    }
    if (output.textContent.includes("❌ Error:")) setStatus("Falló", "error");
    else setStatus("Terminado", "done");
    await refreshProject();
  } catch (error) {
    output.textContent += `\n❌ ${error.message}`;
    setStatus("Error", "error");
  } finally {
    setBusy(false);
  }
}

document.querySelector("#pick-folder").addEventListener("click", () => choosePath("/api/pick-folder", projectPathInput));
document.querySelector("#pick-key").addEventListener("click", () => choosePath("/api/pick-key", keyPathInput));
document.querySelector("#refresh-project").addEventListener("click", refreshProject);
document.querySelector("#clear-output").addEventListener("click", () => { output.textContent = "Salida limpiada."; });
projectPathInput.addEventListener("keydown", (event) => { if (event.key === "Enter") refreshProject(); });
buttons.forEach((button) => button.addEventListener("click", () => runAction(button.dataset.action)));

refreshProject();
