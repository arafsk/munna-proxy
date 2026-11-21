import * as App from "./wailsjs/go/main/App.js";

const $ = (id) => document.getElementById(id);

const state = {
  logSeq: 0,
  hostRunning: false,
  clientRunning: false,
};

function toast(message, isError = false) {
  const el = $("toast");
  el.textContent = message;
  el.classList.toggle("error", isError);
  el.classList.add("show");
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => el.classList.remove("show"), 4000);
}

function errorText(err) {
  if (typeof err === "string") return err;
  if (err && err.message) return err.message;
  return String(err);
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
  } catch (_) {
    const area = document.createElement("textarea");
    area.value = text;
    document.body.appendChild(area);
    area.select();
    document.execCommand("copy");
    area.remove();
  }
  toast("Copied");
}

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key === "text") node.textContent = value;
    else if (key === "class") node.className = value;
    else node.setAttribute(key, value);
  }
  for (const child of children) node.append(child);
  return node;
}

function copyLine(value) {
  return el("div", { class: "copy-line" }, [
    el("code", { text: value }),
    el("button", { type: "button", class: "small secondary", text: "Copy" }, []),
  ]);
}

function withCopy(value) {
  const line = copyLine(value);
  line.querySelector("button").addEventListener("click", () => copyText(value));
  return line;
}

function setPill(id, label, on) {
  const pill = $(id);
  pill.textContent = label;
  pill.classList.toggle("on", on);
  pill.classList.toggle("off", !on);
}

/* ---------- tabs ---------- */

document.querySelectorAll(".tab").forEach((tab) => {
  tab.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach((t) => t.classList.toggle("active", t === tab));
    document.querySelectorAll(".panel").forEach((panel) => {
      panel.classList.toggle("active", panel.id === `tab-${tab.dataset.tab}`);
    });
  });
});

/* ---------- host ---------- */

function hostOptions() {
  return {
    name: $("host-name").value.trim(),
    listen: $("host-listen").value.trim(),
    advertise: $("host-advertise").value.trim(),
    user: $("host-user").value.trim(),
    password: $("host-password").value,
    noAuth: $("host-noauth").checked,
    disableMDNS: !$("host-mdns").checked,
  };
}

function renderHost(info) {
  state.hostRunning = info.running;
  setPill("pill-host", info.running ? "Host on" : "Host off", info.running);
  $("host-start").disabled = info.running;
  $("host-stop").disabled = !info.running;
  for (const id of ["host-name", "host-listen", "host-advertise", "host-user", "host-password", "host-noauth", "host-mdns"]) {
    $(id).disabled = info.running;
  }

  const card = $("host-status");
  card.replaceChildren(el("h2", { text: "Host status" }));
  if (!info.running) {
    card.append(el("p", { class: "empty", text: "Not running." }));
    return;
  }

  const rows = [
    ["Listening", info.listen],
    ["Advertised", info.advertise],
    ["Username", info.noAuth ? "(none)" : info.user],
    ["Password", info.noAuth ? "(none)" : info.password],
    ["mDNS", info.mdns ? "on" : "off"],
    ["Started", new Date(info.started).toLocaleTimeString()],
  ];
  const dl = el("dl", { class: "kv" });
  for (const [key, value] of rows) {
    dl.append(el("dt", { text: key }), el("dd", { text: value }));
  }

  const invite = el("div", {}, [el("div", { class: "hint", text: "Invite (for the Client tab on another device)" })]);
  invite.append(withCopy(info.invite));

  const children = [dl, invite];
  if (info.qr) {
    children.push(el("img", { class: "qr", src: info.qr, alt: "Invite QR code" }));
  }
  card.append(...children);
}

async function refreshHost() {
  try {
    renderHost(await App.HostStatus());
  } catch (err) {
    toast(errorText(err), true);
  }
}

$("host-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  $("host-start").disabled = true;
  try {
    renderHost(await App.StartHost(hostOptions()));
    toast("Host started");
  } catch (err) {
    toast(errorText(err), true);
    $("host-start").disabled = false;
  }
});

$("host-stop").addEventListener("click", async () => {
  $("host-stop").disabled = true;
  try {
    await App.StopHost();
    toast("Host stopped");
  } catch (err) {
    toast(errorText(err), true);
  }
  await refreshHost();
});

/* ---------- client ---------- */

function renderClient(info) {
  state.clientRunning = info.running;
  setPill("pill-client", info.running ? "Client on" : "Client off", info.running);
  $("client-start").disabled = info.running;
  $("client-stop").disabled = !info.running;
  for (const id of ["client-host", "client-user", "client-password", "client-listen", "client-discover"]) {
    $(id).disabled = info.running;
  }

  const card = $("client-status");
  card.replaceChildren(el("h2", { text: "Connection status" }));
  if (!info.running) {
    card.append(el("p", { class: "empty", text: "Not connected." }));
    return;
  }

  const dl = el("dl", { class: "kv" });
  for (const [key, value] of [
    ["Host", info.host],
    ["Local proxy", info.listen],
    ["Username", info.user || "(none)"],
    ["Connected", new Date(info.started).toLocaleTimeString()],
  ]) {
    dl.append(el("dt", { text: key }), el("dd", { text: value }));
  }

  const port = info.listen.split(":").pop();
  const usage = el("div", {}, [
    el("div", { class: "hint", text: "Set an app's proxy to SOCKS5 at this address:" }),
    withCopy(info.listen),
    el("div", { class: "hint", text: "Test it from a terminal:" }),
    withCopy(`curl --socks5-hostname ${info.listen} https://ifconfig.me`),
    el("div", { class: "hint", text: "Localhost service on the host, e.g. port 3000:" }),
    withCopy(`curl --socks5-hostname 127.0.0.1:${port} http://127.0.0.1:3000`),
  ]);

  card.append(dl, usage);
}

async function refreshClient() {
  try {
    renderClient(await App.ClientStatus());
  } catch (err) {
    toast(errorText(err), true);
  }
}

$("client-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  $("client-start").disabled = true;
  try {
    const info = await App.StartClient({
      host: $("client-host").value.trim(),
      user: $("client-user").value.trim(),
      password: $("client-password").value,
      listen: $("client-listen").value.trim(),
    });
    localStorage.setItem("munna.client.host", $("client-host").value.trim());
    localStorage.setItem("munna.client.user", $("client-user").value.trim());
    renderClient(info);
    toast("Connected");
  } catch (err) {
    toast(errorText(err), true);
    $("client-start").disabled = false;
  }
});

$("client-stop").addEventListener("click", async () => {
  $("client-stop").disabled = true;
  try {
    await App.StopClient();
    toast("Disconnected");
  } catch (err) {
    toast(errorText(err), true);
  }
  await refreshClient();
});

$("client-discover").addEventListener("click", async () => {
  const list = $("peer-list");
  list.replaceChildren(el("li", { class: "empty", text: "Searching for 3 seconds…" }));
  try {
    const peers = await App.Discover();
    if (!peers || peers.length === 0) {
      list.replaceChildren(el("li", { class: "empty", text: "No hosts found. Enter the address or invite instead." }));
      return;
    }
    list.replaceChildren(
      ...peers.map((peer) => {
        const button = el("button", { type: "button", text: peer.name || peer.address }, []);
        button.append(el("small", { text: peer.address }));
        button.addEventListener("click", () => {
          $("client-host").value = peer.address;
        });
        return el("li", {}, [button]);
      })
    );
  } catch (err) {
    list.replaceChildren();
    toast(errorText(err), true);
  }
});

/* ---------- logs ---------- */

async function pollLogs() {
  try {
    const entries = await App.Logs(state.logSeq);
    if (!entries || entries.length === 0) return;
    const output = $("log-output");
    const nearBottom = output.scrollHeight - output.scrollTop - output.clientHeight < 40;
    for (const entry of entries) {
      state.logSeq = Math.max(state.logSeq, entry.seq);
      output.append(document.createTextNode(entry.message + "\n"));
    }
    if (nearBottom) output.scrollTop = output.scrollHeight;
  } catch (_) {
    // The log is best effort; the status cards report real failures.
  }
}

$("logs-clear").addEventListener("click", () => {
  $("log-output").textContent = "";
});

$("debug").addEventListener("change", async () => {
  localStorage.setItem("munna.debug", $("debug").checked ? "1" : "0");
  await App.SetDebug($("debug").checked);
});

/* ---------- startup ---------- */

async function init() {
  try {
    const defaults = await App.Defaults();
    $("host-name").value = defaults.name;
    $("host-listen").value = defaults.hostListen;
    $("host-user").value = "munna";
    $("client-listen").value = defaults.clientListen;
    $("host-advertise").placeholder = `auto-detected (${defaults.lanAddress}:${defaults.hostPort})`;
  } catch (err) {
    toast(errorText(err), true);
  }

  $("client-host").value = localStorage.getItem("munna.client.host") || "";
  $("client-user").value = localStorage.getItem("munna.client.user") || "";
  $("debug").checked = localStorage.getItem("munna.debug") === "1";
  if ($("debug").checked) await App.SetDebug(true);

  await refreshHost();
  await refreshClient();
  setInterval(refreshHost, 3000);
  setInterval(refreshClient, 3000);
  setInterval(pollLogs, 1000);
}

init();
