const inviteBox = document.getElementById("invite");

async function updateStatus() {
  const r = await fetch("/api/server/status");
  const data = await r.json();

  document.getElementById("status").textContent = data.running
    ? "Running"
    : "Stopped";

  document.getElementById("pid").textContent = data.running ? data.pid : "-";
}

// Format bytes to human readable
function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Format uptime to human readable
function formatUptime(seconds) {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);
  
  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  } else {
    return `${secs}s`;
  }
}

// Load and display metrics
async function loadMetrics() {
  try {
    const r = await fetch("/api/metrics");
    const m = await r.json();
    
    // Bandwidth
    document.getElementById("mbps-out").textContent = m.mbps_sent.toFixed(2);
    document.getElementById("bytes-out").textContent = formatBytes(m.bytes_sent);
    document.getElementById("mbps-in").textContent = m.mbps_received.toFixed(2);
    document.getElementById("bytes-in").textContent = formatBytes(m.bytes_received);
    
    // Connections
    document.getElementById("active-conn").textContent = m.active_connections;
    document.getElementById("total-conn").textContent = `Total: ${m.total_connections}`;
    document.getElementById("players-joined").textContent = m.players_joined;
    document.getElementById("players-left").textContent = m.players_left;
    
    // Uptime
    document.getElementById("uptime").textContent = formatUptime(m.uptime_seconds);
    document.getElementById("uptime-details").textContent = new Date(m.uptime_seconds * 1000).toISOString().substr(11, 8);
    
    // Performance
    document.getElementById("avg-latency").textContent = Math.round(m.avg_latency_ms);
    document.getElementById("peak-latency").textContent = `Peak: ${Math.round(m.peak_latency_ms)} ms`;
    
    // Game State
    document.getElementById("car-updates").textContent = m.car_updates.toLocaleString();
    const failureRate = m.car_update_failures > 0 ? (m.car_failure_rate.toFixed(2)) : "0";
    document.getElementById("car-health").textContent = `Health: ${(100 - parseFloat(failureRate)).toFixed(2)}%`;
    
    // Packets
    document.getElementById("pkt-sent").textContent = m.packets_sent.toLocaleString();
    document.getElementById("pkt-recv").textContent = m.packets_received.toLocaleString();
    const avgSize = m.packets_sent > 0 ? Math.round(m.avg_packet_size_out) : 0;
    document.getElementById("avg-pkt").textContent = avgSize;
    
  } catch (e) {
    // Server metrics not available
  }
}

// Reset metrics
async function resetMetrics() {
  try {
    await fetch("/api/metrics/reset", { method: "POST" });
    setTimeout(loadMetrics, 500);
  } catch (e) {
    console.log("Error resetting metrics: " + e);
  }
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
    let sessionData = JSON.parse(data.session);

    inviteBox.textContent = data.invite || "-";

    const select = document.getElementById("trackSelect");
    const selectSession = document.getElementById("trackSelectSession");

    if(!sessionData.switchingSession || selectSession.children.length == 0) {
      select.innerHTML = "";
      selectSession.innerHTML = "";
      data.tracks.forEach((name) => {
        const opt = document.createElement("option");
        opt.value = name;
        opt.textContent = name;

        if (name === data.current) opt.selected = true;

        const opt2 = document.createElement("option");
        opt2.value = name;
        opt2.textContent = name;

        select.appendChild(opt);
        selectSession.appendChild(opt2);
      });
    }
    let sessionInfoDiv = document.getElementById("sessionInfo")
    sessionInfoDiv.innerHTML = `
      <p>Session ID: <strong>${sessionData["sessionId"]}</strong></p>
      <p>Session Gamemode: <strong>${sessionData["gamemode"] == 1 ? "Competitive" : "Casual"}</strong></p>
      <p>Max players: <strong>${sessionData["maxPlayers"]}</strong></p>
      <p>Switching sessions? <strong>${sessionData["switchingSession"] ? "Yes" : "No"}</strong></p>
      `;
      document.getElementById("startSessionBtn").disabled = !sessionData["switchingSession"]
      document.getElementById("sendSessionBtn").disabled = !sessionData["switchingSession"]
      document.getElementById("endSessionBtn").disabled = sessionData["switchingSession"]

  } catch(e) {
    console.log("Error " + e)
    inviteBox.textContent = "(server not running)";
  }
}

async function endSession() {
  const r = await fetch("/api/session/end", { method: "POST" });
  await loadServerData()
}
async function startSession() {
  const r = await fetch("/api/session/start", { method: "POST" });
  await loadServerData()
}

async function sendSession() {
  let index = 0;
  for(let child of document.getElementById("gamemodePicker").children) {
    if(child.children[0].checked) break;
    index++;
  }
  console.log(JSON.stringify({ 
      gamemode: index, 
      track: document.getElementById("trackSelectSession").value,
      maxPlayers: document.getElementById("maxPlayers").value,
    }))
  await fetch("/api/session/set", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ 
      gamemode: index, 
      track: document.getElementById("trackSelectSession").value,
      maxPlayers: parseInt(document.getElementById("maxPlayers").value),
    }),
  });
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
        <td><button class="uk-button uk-button-danger" type="button" onclick="kickPlayer(${p.id})">Kick</button></td>
      `;

      tbody.appendChild(tr);
    });
  } catch {
    // server not running
  }
}

async function kickPlayer(id) {
  await fetch("/api/kick", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id }),
  });
}

// ---------- INIT ----------

function main() {
  updateStatus();
  loadServerData();
  loadPlayers();
  loadMetrics();

  setInterval(updateStatus, 2000);
  setInterval(loadPlayers, 1000);
  setInterval(loadServerData, 3000);
  setInterval(loadMetrics, 1000);  // Update metrics every second for real-time view
}

main();
