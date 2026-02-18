const inviteBox = document.getElementById("invite");

async function updateStatus() {
  const r = await fetch("/api/server/status");
  const data = await r.json();

  document.getElementById("status").textContent = data.running
    ? "Running"
    : "Stopped";

  document.getElementById("pid").textContent = data.running ? data.pid : "-";
}

async function startServer() {
  await fetch("/api/server/start", { method: "POST" });

  setTimeout(() => {
    updateStatus();
    loadServerData();
  }, 800);
}

async function stopServer() {
  await fetch("/api/server/stop", { method: "POST" });
  setTimeout(updateStatus, 500);
}

// ---------- INVITE + TRACKS ----------

async function loadServerData() {
  try {
    const r = await fetch("/api/tracks");
    const data = await r.json();

    inviteBox.textContent = data.invite || "-";

    const select = document.getElementById("trackSelect");
    select.innerHTML = "";

    data.tracks.forEach((name) => {
      const opt = document.createElement("option");
      opt.value = name;
      opt.textContent = name;

      if (name === data.current) opt.selected = true;

      select.appendChild(opt);
    });
  } catch {
    inviteBox.textContent = "(server not running)";
  }
}

async function createInvite() {
  const r = await fetch("/api/invite", { method: "POST" });
  const data = await r.json();

  inviteBox.textContent = data.invite;
  await loadServerData();
}

async function setTrack() {
  const name = document.getElementById("trackSelect").value;

  await fetch("/api/tracks", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}

// ---------- PLAYERS ----------

async function loadPlayers() {
  try {
    const r = await fetch("/api/players");
    const data = await r.json();

    const tbody = document.querySelector("#players tbody");
    tbody.innerHTML = "";

    data.players.forEach((p) => {
      const tr = document.createElement("tr");

      tr.innerHTML = `
        <td>${p.name}</td>
        <td>${p.time}</td>
        <td>${p.ping} ms</td>
      `;

      tbody.appendChild(tr);
    });
  } catch {
    // server not running
  }
}

// ---------- INIT ----------

function main() {
  updateStatus();
  loadServerData();
  loadPlayers();

  setInterval(updateStatus, 2000);
  setInterval(loadPlayers, 1000);
  setInterval(loadServerData, 3000);
}

main();
