const root = document.querySelector("[data-json-tool]");

if (root) {
  const input = root.querySelector("[data-json-input]");
  const output = root.querySelector("[data-json-output]");
  const tree = root.querySelector("[data-json-tree]");
  const status = root.querySelector("[data-json-status]");
  const maxBytes = 1024 * 1024;
  const maxDepth = 20;
  const maxNodes = 5000;

  function setStatus(message, isError = false) {
    status.textContent = message;
    status.dataset.state = isError ? "error" : "success";
  }

  function parseInput() {
    if (!input.value.trim()) {
      setStatus(root.dataset.needInput, true);
      return null;
    }
    if (new TextEncoder().encode(input.value).byteLength > maxBytes) {
      setStatus(root.dataset.inputTooLarge, true);
      return null;
    }
    try {
      return JSON.parse(input.value);
    } catch (error) {
      const position = /position (\d+)/.exec(error.message);
      if (position) {
        const prefix = input.value.slice(0, Number(position[1]));
        const line = prefix.split("\n").length;
        const column = prefix.length - prefix.lastIndexOf("\n");
        setStatus(`${root.dataset.invalidJson} ${line}:${column}`, true);
      } else {
        setStatus(root.dataset.invalidJson, true);
      }
      return null;
    }
  }

  function valueText(value) {
    if (typeof value === "string") return JSON.stringify(value);
    if (value === null) return "null";
    return String(value);
  }

  function renderTree(value) {
    let nodes = 0;
    let truncated = false;
    tree.replaceChildren();

    function appendValue(parent, current, name, depth) {
      if (nodes >= maxNodes || depth > maxDepth) {
        truncated = true;
        return;
      }
      nodes += 1;
      const isContainer = current !== null && typeof current === "object";
      const item = document.createElement("div");
      item.className = "json-tree-item";

      if (!isContainer) {
        const text = document.createElement("span");
        text.className = `json-value json-${current === null ? "null" : typeof current}`;
        text.textContent = `${name}: ${valueText(current)}`;
        item.append(text);
        parent.append(item);
        return;
      }

      const details = document.createElement("details");
      details.open = depth < 2;
      const summary = document.createElement("summary");
      const count = Array.isArray(current) ? current.length : Object.keys(current).length;
      summary.textContent = `${name}: ${Array.isArray(current) ? "Array" : "Object"} (${count})`;
      details.append(summary);
      const children = document.createElement("div");
      children.className = "json-tree-children";
      for (const [key, child] of Object.entries(current)) {
        appendValue(children, child, Array.isArray(current) ? `[${key}]` : key, depth + 1);
        if (truncated) break;
      }
      details.append(children);
      item.append(details);
      parent.append(item);
    }

    appendValue(tree, value, "root", 0);
    if (truncated) setStatus(root.dataset.treeTruncated, false);
  }

  function transform(indentation) {
    const value = parseInput();
    if (value === null && input.value.trim() !== "null") return;
    output.value = JSON.stringify(value, null, indentation);
    setStatus("");
    renderTree(value);
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

  root.querySelector("[data-json-action='format']").addEventListener("click", () => transform(2));
  root.querySelector("[data-json-action='minify']").addEventListener("click", () => transform(0));
  root.querySelector("[data-json-copy]").addEventListener("click", copyResult);
  root.querySelector("[data-json-clear]").addEventListener("click", () => {
    input.value = "";
    output.value = "";
    tree.replaceChildren();
    setStatus(root.dataset.cleared);
    input.focus();
  });
}
