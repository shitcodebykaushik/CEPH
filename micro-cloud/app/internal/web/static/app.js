const API = '/api/v1';
let tenants = [], volumes = [], workspaces = [];
let requestedId = null;
const IS_PORTAL = window.location.pathname === '/portal';

async function api(method, path, body) {
  const opts = { method, headers: {} };
  if (body) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  const r = await fetch(API + path, opts);
  if (!r.ok) { const e = await r.json().catch(()=>({error:r.statusText})); throw new Error(e.error||r.statusText); }
  return r.status===204 ? null : r.json();
}

if (!IS_PORTAL) {
  document.querySelectorAll('.sidebar li').forEach(el => {
    el.addEventListener('click', () => {
      document.querySelectorAll('.sidebar li').forEach(l=>l.classList.remove('active'));
      el.classList.add('active');
      document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
      const page = document.getElementById('page-'+el.dataset.page);
      if (page) { page.classList.add('active'); renderPage(el.dataset.page); }
    });
  });
}

function renderPage(name) {
  if (name==='landing') renderLanding();
  else if (name==='dashboard') renderDashboard();
  else if (name==='workspaces') renderWorkspaces();
  else if (name==='tenants') renderTenants();
  else if (name==='volumes') renderVolumes();
}

function showModal(title, bodyHTML) {
  document.getElementById('modal-title').textContent = title;
  document.getElementById('modal-body').innerHTML = bodyHTML;
  document.getElementById('modal-overlay').classList.remove('hidden');
  document.getElementById('modal').classList.remove('hidden');
}
function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden');
  document.getElementById('modal').classList.add('hidden');
}

async function loadAll() {
  try { tenants = await api('GET','/tenants'); } catch(e) { tenants = []; }
  try { volumes = await api('GET','/volumes'); } catch(e) { volumes = []; }
  try { workspaces = await api('GET','/workspaces'); } catch(e) { workspaces = []; }
}

// ─── Landing: Request a Workspace ─────────────────────────────
async function renderLanding() {
  requestedId = null;
  document.getElementById('page-landing').innerHTML = `
    <div class="hero">
      <h1>☁️ Micro-Cloud Workspace Portal</h1>
      <p>Request a private workspace with persistent storage. Provisioned instantly and accessible via SSH.</p>
    </div>
    <div id="request-form-area">
      <div class="request-form">
        <h2>Request a Workspace</h2>
        <p class="subtitle">Fill in your details and we'll spin up a workspace for you.</p>
        <div class="form-group"><label>Your Name</label><input id="f-req-name" placeholder="e.g. Alice"></div>
        <div class="form-group"><label>Email Address</label><input id="f-req-email" type="email" placeholder="alice@example.com"></div>
        <div class="form-group"><label>Base Image</label>
          <select id="f-req-image">
            <option value="alpine:latest">Alpine Linux (small)</option>
            <option value="ubuntu:22.04">Ubuntu 22.04</option>
            <option value="debian:bookworm-slim">Debian Bookworm</option>
          </select></div>
        <div class="form-group"><label>Volume Size</label>
          <select id="f-req-size">
            <option value="256">256 MB</option>
            <option value="512" selected>512 MB</option>
            <option value="1024">1 GB</option>
            <option value="5120">5 GB</option>
            <option value="10240">10 GB</option>
          </select></div>
        <div class="form-actions">
          <button class="btn btn-primary" onclick="submitRequest()" style="width:100%;justify-content:center;padding:12px">
            🚀 Request Workspace
          </button>
        </div>
      </div>
    </div>
    <div id="request-result" class="hidden"></div>
  `;
}

async function submitRequest() {
  const name = document.getElementById('f-req-name').value.trim();
  const email = document.getElementById('f-req-email').value.trim();
  const image = document.getElementById('f-req-image').value;
  const size = parseInt(document.getElementById('f-req-size').value);
  if (!name) return alert('Please enter your name');
  if (!email) return alert('Please enter your email');

  const btn = document.querySelector('#request-form-area .btn-primary');
  btn.disabled = true; btn.textContent = '⏳ Provisioning...';

  try {
    const resp = await api('POST','/workspaces/request',{name,email,image,size_mb:size});
    requestedId = resp.workspace_id;
    document.getElementById('request-form-area').classList.add('hidden');
    showRequestResult(resp);
    pollWorkspace(requestedId);
  } catch(e) {
    btn.disabled = false; btn.textContent = '🚀 Request Workspace';
    alert('Failed: ' + e.message);
  }
}

function showRequestResult(resp) {
  const area = document.getElementById('request-result');
  area.classList.remove('hidden');
  area.innerHTML = `
    <div class="result-card">
      <div class="icon">✅</div>
      <h2>Workspace Provisioning</h2>
      <p>Your workspace is being created. It will be ready in a few seconds.</p>
      <div class="detail">
        <div><strong>Workspace ID</strong><span class="val">${resp.workspace_id}</span></div>
        <div style="margin-top:8px"><strong>Status</strong><span class="val" id="req-status">${resp.status}</span></div>
      </div>
      <div id="req-access-info"></div>
      <button class="btn" onclick="resetLanding()" style="margin-top:8px">← Request Another</button>
    </div>
  `;
}

async function pollWorkspace(id) {
  for (let i = 0; i < 30; i++) {
    await new Promise(r => setTimeout(r, 2000));
    try {
      const w = await api('GET',`/workspaces/${id}`);
      const statusEl = document.getElementById('req-status');
      if (statusEl) statusEl.textContent = w.status;
      if (w.status === 'running') {
        const info = document.getElementById('req-access-info');
        if (info) {
          info.innerHTML = `
            <div class="ssh-box"><code>ssh -p ${w.port} root@localhost</code>
            <button class="copy" onclick="copySSH(${w.port})">📋 Copy</button></div>
            <p style="margin-top:8px"><a class="btn btn-primary" href="/terminal?id=${w.id}" target="_blank" style="text-decoration:none">
              🖥️ Open Browser Terminal
            </a></p>
            <p style="font-size:13px;color:var(--text2);margin-top:4px">Container: ${short(w.container_id)}</p>
          `;
        }
        break;
      }
      if (w.status === 'failed') break;
    } catch(e) {}
  }
}

function resetLanding() {
  renderLanding();
}

function copySSH(port) {
  navigator.clipboard.writeText(`ssh -p ${port} root@localhost`);
}

// ─── Dashboard (Admin) ────────────────────────────────────────
async function renderDashboard() {
  await loadAll();
  const running = workspaces.filter(w=>w.status==='running').length;
  const totalVol = volumes.filter(v=>v.status!=='deleted').length;
  const totalTen = tenants.length;
  const failed = workspaces.filter(w=>w.status==='failed').length;

  document.getElementById('page-dashboard').innerHTML = `
    <h1>Dashboard</h1>
    <p class="subtitle">Overview of your Micro-Cloud infrastructure</p>
    <div class="stats">
      <div class="stat-card"><div class="label">Tenants</div><div class="value blue">${totalTen}</div></div>
      <div class="stat-card"><div class="label">Running Workspaces</div><div class="value green">${running}</div></div>
      <div class="stat-card"><div class="label">Volumes</div><div class="value yellow">${totalVol}</div></div>
      <div class="stat-card"><div class="label">Failed</div><div class="value red">${failed}</div></div>
    </div>

    <h2>Recent Workspaces</h2>
    ${workspaces.length
      ? renderTable(['Workspace','Requester','Image','Status','SSH','Created'],
          workspaces.slice(0,8).map(w=>[
            `<a href="#" onclick="event.preventDefault();showWorkspace('${w.id}')">${short(w.id)}</a>`,
            w.requester_name || short(w.tenant_id),
            w.image,
            `<span class="badge badge-${w.status}">${w.status}</span>`,
            w.status==='running' ? `<code>localhost:${w.port}</code>` : '—',
            w.created_at
          ]))
      : '<div class="empty"><p>No workspaces yet.</p></div>'}

    <h2>Quick Actions</h2>
    <div style="display:flex;gap:8px;flex-wrap:wrap">
      <button class="btn" onclick="switchPage('tenants');showCreateTenant()">+ New Tenant</button>
      <button class="btn" onclick="switchPage('workspaces')">📋 All Workspaces</button>
    </div>
  `;
}

// ─── Tenants ──────────────────────────────────────────────────
async function renderTenants() {
  await loadAll();
  document.getElementById('page-tenants').innerHTML = `
    <div class="toolbar"><div><h1>Tenants</h1><p class="subtitle">Organizations and their resource quotas</p></div>
      <button class="btn btn-primary" onclick="showCreateTenant()">+ New Tenant</button></div>
    ${tenants.length
      ? renderTable(['Name','ID','Active Instances','Storage Quota','Created'], tenants.map(t=>[
          esc(t.name), short(t.id), t.active_instances,
          formatBytes(t.storage_quota_bytes), t.created_at
        ]))
      : '<div class="empty"><p>No tenants yet. <a href="#" onclick="showCreateTenant()">Create one</a>.</p></div>'}
  `;
}

function showCreateTenant() {
  showModal('New Tenant', `
    <div class="form-group"><label>Name</label><input id="f-tenant-name" placeholder="e.g. Acme Corp"></div>
    <div class="form-group"><label>Storage Quota (GB)</label><input id="f-tenant-quota" type="number" value="10" min="1"></div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Cancel</button>
      <button class="btn btn-primary" onclick="createTenant()">Create Tenant</button>
    </div>
  `);
}
async function createTenant() {
  const name = document.getElementById('f-tenant-name').value.trim();
  if (!name) return alert('Name is required');
  const quota = parseInt(document.getElementById('f-tenant-quota').value) * 1073741824;
  await api('POST','/tenants',{name,storage_quota_bytes:quota});
  closeModal(); renderTenants();
}

// ─── Volumes ──────────────────────────────────────────────────
async function renderVolumes() {
  await loadAll();
  document.getElementById('page-volumes').innerHTML = `
    <div class="toolbar"><div><h1>Volumes</h1><p class="subtitle">Persistent storage attached to workspaces</p></div>
      <button class="btn btn-primary" onclick="showCreateVolume()">+ New Volume</button></div>
    ${volumes.length
      ? renderTable(['ID','Tenant','Size','Status','Mount Path','Created'], volumes.map(v=>[
          short(v.id), short(v.tenant_id),
          formatMB(v.size_mb),
          `<span class="badge badge-${v.status}">${v.status}</span>`,
          v.mount_path ? short(v.mount_path) : '—',
          v.created_at
        ]))
      : '<div class="empty"><p>No volumes yet.</p></div>'}
  `;
}

function showCreateVolume() {
  if (!tenants.length) return alert('Create a tenant first');
  showModal('New Volume', `
    <div class="form-group"><label>Tenant</label>
      <select id="f-vol-tenant">${tenants.map(t=>`<option value="${t.id}">${esc(t.name)}</option>`).join('')}</select></div>
    <div class="form-group"><label>Size (MB)</label><input id="f-vol-size" type="number" value="1024" min="128"></div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Cancel</button>
      <button class="btn btn-primary" onclick="createVolume()">Create Volume</button>
    </div>
  `);
}
async function createVolume() {
  const tenant_id = document.getElementById('f-vol-tenant').value;
  const size_mb = parseInt(document.getElementById('f-vol-size').value);
  await api('POST','/volumes',{tenant_id,size_mb});
  closeModal(); renderVolumes();
}

// ─── Workspaces (Admin) ───────────────────────────────────────
async function renderWorkspaces() {
  await loadAll();
  document.getElementById('page-workspaces').innerHTML = `
    <div class="toolbar"><div><h1>Workspaces</h1><p class="subtitle">All compute workspaces — track who has access</p></div>
      <span style="color:var(--text2);font-size:14px">Users request workspaces at <a href="/portal">/portal</a></span></div>
    <div class="tabs">
      <button class="tab active" data-tab="all" onclick="switchTab(event,'all')">All</button>
      <button class="tab" data-tab="running" onclick="switchTab(event,'running')">Running</button>
      <button class="tab" data-tab="failed" onclick="switchTab(event,'failed')">Failed</button>
    </div>
    <div id="ws-tab-all" class="tab-content active">${buildWsTable(workspaces)}</div>
    <div id="ws-tab-running" class="tab-content">${buildWsTable(workspaces.filter(w=>w.status==='running'))}</div>
    <div id="ws-tab-failed" class="tab-content">${buildWsTable(workspaces.filter(w=>w.status==='failed'))}</div>
  `;
}

function buildWsTable(list) {
  if (!list.length) return '<div class="empty"><p>No workspaces in this view.</p></div>';
  return renderTable(
    ['Workspace','Requester','Email','Tenant','Image','Status','SSH','Created','Actions'],
    list.map(w=>[
      `<a href="#" onclick="event.preventDefault();showWorkspace('${w.id}')">${short(w.id)}</a>`,
      w.requester_name || '—',
      w.requester_email || '—',
      short(w.tenant_id),
      w.image,
      `<span class="badge badge-${w.status}">${w.status}</span>`,
      w.status==='running' ? `<code>localhost:${w.port}</code>` : '—',
      w.created_at,
      w.status==='running'
        ? `<button class="btn btn-sm btn-danger" onclick="deleteWorkspace('${w.id}')">Terminate</button>`
        : w.status==='failed' || w.status==='stopped'
          ? `<button class="btn btn-sm btn-danger" onclick="deleteWorkspace('${w.id}')">Remove</button>`
          : '<span class="badge badge-launching">pending</span>'
    ])
  );
}

function switchTab(e, name) {
  document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
  e.currentTarget.classList.add('active');
  document.querySelectorAll('.tab-content').forEach(t=>t.classList.remove('active'));
  document.getElementById('ws-tab-'+name).classList.add('active');
}

async function deleteWorkspace(id) {
  if (!confirm('Terminate this workspace? All associated data will be lost.')) return;
  await api('DELETE',`/workspaces/${id}`);
  renderWorkspaces();
}

async function showWorkspace(id) {
  let w;
  try { w = await api('GET',`/workspaces/${id}`); } catch(e) { return alert('Workspace not found'); }
  showModal('Workspace Details', `
    <div class="detail-grid">
      <div class="detail-item"><div class="label">Workspace ID</div><div class="val">${w.id}</div></div>
      <div class="detail-item"><div class="label">Status</div><div class="val"><span class="badge badge-${w.status}">${w.status}</span></div></div>
      <div class="detail-item"><div class="label">Image</div><div class="val">${w.image}</div></div>
      <div class="detail-item"><div class="label">Tenant</div><div class="val">${short(w.tenant_id)}</div></div>
      <div class="detail-item"><div class="label">Requester</div><div class="val">${w.requester_name || '—'}</div></div>
      <div class="detail-item"><div class="label">Email</div><div class="val">${w.requester_email || '—'}</div></div>
      <div class="detail-item"><div class="label">Volume ID</div><div class="val">${w.volume_id || '—'}</div></div>
      <div class="detail-item"><div class="label">Container</div><div class="val">${short(w.container_id) || '—'}</div></div>
      <div class="detail-item"><div class="label">Port</div><div class="val">${w.port || '—'}</div></div>
      <div class="detail-item"><div class="label">Created</div><div class="val">${w.created_at}</div></div>
      ${w.status==='running' ? `<div class="detail-item full"><div class="label">SSH Access</div>
        <div class="ssh-box"><code>ssh -p ${w.port} root@localhost</code>
        <button class="copy" onclick="copySSH(${w.port})">📋 Copy</button></div>
        <div style="margin-top:8px"><a class="btn btn-primary" href="/terminal?id=${w.id}" target="_blank" style="text-decoration:none;width:100%;justify-content:center">
          🖥️ Open Browser Terminal
        </a></div></div>` : ''}
    </div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Close</button>
      ${w.status==='running' ? `<button class="btn btn-danger" onclick="closeModal();deleteWorkspace('${w.id}')">Terminate</button>` : ''}
    </div>
  `);
}

// ─── Navigation helper ───────────────────────────────────────
function switchPage(name) {
  document.querySelectorAll('.sidebar li').forEach(l=>l.classList.remove('active'));
  document.querySelector(`.sidebar li[data-page="${name}"]`)?.classList.add('active');
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  const page = document.getElementById('page-'+name);
  if (page) { page.classList.add('active'); renderPage(name); }
}

// ─── Helpers ──────────────────────────────────────────────────
function renderTable(headers, rows) {
  return `<div class="table-wrap"><table>
    <thead><tr>${headers.map(h=>`<th>${h}</th>`).join('')}</tr></thead>
    <tbody>${rows.map(r=>`<tr>${r.map(c=>`<td>${c||'—'}</td>`).join('')}</tr>`).join('')}</tbody>
  </table></div>`;
}
function short(s){return s?s.length>12?s.slice(0,12)+'…':s:'—'}
function esc(s){const d=document.createElement('div');d.textContent=s;return d.innerHTML}
function formatBytes(b){if(!b)return'—';const u=['B','KB','MB','GB','TB'];let i=0;let n=b;while(n>=1024&&i<4){n/=1024;i++}return n.toFixed(1)+' '+u[i]}
function formatMB(m){if(!m)return'—';if(m>=1024)return(m/1024).toFixed(1)+' GB';return m+' MB'}

if (IS_PORTAL) {
  renderPage('landing');
} else {
  renderPage('dashboard');
}
