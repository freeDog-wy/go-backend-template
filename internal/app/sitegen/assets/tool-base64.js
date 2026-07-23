const root = document.querySelector("[data-base64-tool]");

if (root) {
  const input = root.querySelector("[data-base64-input]");
  const output = root.querySelector("[data-base64-output]");
  const status = root.querySelector("[data-base64-status]");
  const maxBytes = 1024 * 1024;

  function setStatus(message, isError = false) {
    status.textContent = message;
    status.dataset.state = isError ? "error" : "success";
  }

  function inputIsValid() {
    if (!input.value) {
      setStatus(root.dataset.needInput, true);
      return false;
    }
    if (new TextEncoder().encode(input.value).byteLength > maxBytes) {
      setStatus(root.dataset.inputTooLarge, true);
      return false;
    }
    return true;
  }

  function encodeBase64(value) {
    const bytes = new TextEncoder().encode(value);
    let binary = "";
    for (let index = 0; index < bytes.length; index += 1) {
      binary += String.fromCharCode(bytes[index]);
    }
    return btoa(binary);
  }

  function decodeBase64(value) {
    let normalized = value.replace(/\s/g, "").replace(/-/g, "+").replace(/_/g, "/");
    if (!normalized || /[^A-Za-z0-9+/=]/.test(normalized)) throw new Error("invalid base64");
    const firstPadding = normalized.indexOf("=");
    if (firstPadding !== -1 && /[^=]/.test(normalized.slice(firstPadding))) throw new Error("invalid padding");
    normalized = normalized.replace(/=+$/, "");
    if (normalized.length % 4 === 1) throw new Error("invalid length");
    normalized += "=".repeat((4 - (normalized.length % 4)) % 4);

    const binary = atob(normalized);
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  }

  async function copyResult() {
    if (!output.value) {
      setStatus(root.dataset.needInput, true);
      return;
    }
    try {
      await navigator.clipboard.writeText(output.value);
      setStatus(root.dataset.copied);
    } catch {
      output.focus();
      output.select();
      setStatus(root.dataset.copyFailed, true);
    }
  }

  root.querySelector("[data-base64-action='encode']").addEventListener("click", () => {
    if (!inputIsValid()) return;
    output.value = encodeBase64(input.value);
    setStatus("");
  });

  root.querySelector("[data-base64-action='decode']").addEventListener("click", () => {
    if (!inputIsValid()) return;
    try {
      output.value = decodeBase64(input.value);
      setStatus("");
    } catch {
      output.value = "";
      setStatus(root.dataset.invalidBase64, true);
    }
  });

  root.querySelector("[data-base64-copy]").addEventListener("click", copyResult);
  root.querySelector("[data-base64-clear]").addEventListener("click", () => {
    input.value = "";
    output.value = "";
    setStatus(root.dataset.cleared);
    input.focus();
  });
}
