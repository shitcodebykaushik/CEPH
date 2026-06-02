const API = '/api/v1';

let tenants = [], volumes = [], workspaces = [];

async function api(method, path, body) {
  const opts = { method, headers: {} };
  if (body) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  const r = await fetch(API + path, opts);
  if (!r.ok) { const e = await r.json().catch(()=>({error:r.statusText})); throw new Error(e.error||r.statusText); }
  return r.status===204 ? null : r.json();
}

// ─── Routing ───────────────────────────────────────────────────
document.querySelectorAll('.sidebar li').forEach(el => {
  el.addEventListener('click', () => {
    document.querySelectorAll('.sidebar li').forEach(l=>l.classList.remove('active'));
    el.classList.add('active');
    document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
    const page = document.getElementById('page-'+el.dataset.page);
    if (page) { page.classList.add('active'); renderPage(el.dataset.page); }
  });
});

function renderPage(name) {
  if (name==='dashboard') renderDashboard();
  else if (name==='tenants') renderTenants();
  else if (name==='volumes') renderVolumes();
  else if (name==='workspaces') renderWorkspaces();
}

// ─── Modal ─────────────────────────────────────────────────────
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

// ─── Load all data ─────────────────────────────────────────────
async function loadAll() {
  try { tenants = await api('GET','/tenants'); } catch(e) { tenants = []; }
  try { volumes = await api('GET','/volumes'); } catch(e) { volumes = []; }
  try { workspaces = await api('GET','/workspaces'); } catch(e) { workspaces = []; }
}

// ─── Dashboard ──────────────────────────────────────────────────
async function renderDashboard() {
  await loadAll();
  const running = workspaces.filter(w=>w.status==='running').length;
  const totalVol = volumes.filter(v=>v.status!=='deleted').length;
  const totalTen = tenants.length;
  document.getElementById('page-dashboard').innerHTML = `
    <h1>Dashboard</h1>
    <p class="subtitle">Overview of your Micro-Cloud infrastructure</p>
    <div class="stats">
      <div class="stat-card"><div class="label">Tenants</div><div class="value blue">${totalTen}</div></div>
      <div class="stat-card"><div class="label">Active Workspaces</div><div class="value green">${running}</div></div>
      <div class="stat-card"><div class="label">Volumes</div><div class="value yellow">${totalVol}</div></div>
      <div class="stat-card"><div class="label">Ceph Cluster</div><div class="value">${await cephStatus()}</div></div>
    </div>
    <h2>Recent Workspaces</h2>
    ${workspaces.length ? renderTable(['Workspace','Tenant','Image','Status'], workspaces.slice(0,5).map(w=>[
      `<a href="#" onclick="event.preventDefault();showWorkspace('${w.id}')">${short(w.id)}</a>`,
      short(w.tenant_id), w.image,
      `<span class="badge badge-${w.status}">${w.status}</span>`
    ])) : '<div class="empty"><p>No workspaces yet. Create one from the Workspaces page.</p></div>'}
    <h2 style="margin-top:24px">Quick Actions</h2>
    <div style="display:flex;gap:8px;flex-wrap:wrap">
      <button class="btn btn-primary" onclick="showCreateTenant()">+ New Tenant</button>
      <button class="btn btn-primary" onclick="showCreateWorkspace()">+ Launch Workspace</button>
    </div>
  `;
}

async function cephStatus() {
  try {
    const r = await fetch('/health');
    const d = await r.json();
    return d.status==='ok' ? '✅ Connected' : '⚠️ Error';
  } catch(e) { return '❌ Offline'; }
}

// ─── Tenants ────────────────────────────────────────────────────
async function renderTenants() {
  await loadAll();
  document.getElementById('page-tenants').innerHTML = `
    <div class="toolbar"><div><h1>Tenants</h1><p class="subtitle">Manage your tenants and their quotas</p></div>
      <button class="btn btn-primary" onclick="showCreateTenant()">+ New Tenant</button></div>
    ${tenants.length
      ? renderTable(['Name','ID','Active Instances','Storage Quota','Created'], tenants.map(t=>[
          esc(t.name), short(t.id), t.active_instances,
          formatBytes(t.storage_quota_bytes), t.created_at
        ]))
      : '<div class="empty"><p>No tenants yet.</p></div>'}
  `;
}

function showCreateTenant() {
  showModal('New Tenant', `
    <div class="form-group"><label>Name</label><input id="f-tenant-name" placeholder="e.g. Alice"></div>
    <div class="form-group"><label>Storage Quota (GB)</label><input id="f-tenant-quota" type="number" value="10"></div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Cancel</button>
      <button class="btn btn-primary" onclick="createTenant()">Create</button>
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

// ─── Volumes ────────────────────────────────────────────────────
async function renderVolumes() {
  await loadAll();
  document.getElementById('page-volumes').innerHTML = `
    <div class="toolbar"><div><h1>Volumes</h1><p class="subtitle">Block storage volumes backed by Ceph RBD</p></div>
      <button class="btn btn-primary" onclick="showCreateVolume()">+ New Volume</button></div>
    ${volumes.length
      ? renderTable(['ID','Tenant','Size','Status','Image','Created'], volumes.map(v=>[
          short(v.id), short(v.tenant_id),
          formatMB(v.size_mb),
          `<span class="badge badge-${v.status}">${v.status}</span>`,
          v.image_name, v.created_at
        ]))
      : '<div class="empty"><p>No volumes yet.</p></div>'}
  `;
}

function showCreateVolume() {
  showModal('New Volume', `
    <div class="form-group"><label>Tenant</label>
      <select id="f-vol-tenant">${tenants.map(t=>`<option value="${t.id}">${esc(t.name)}</option>`).join('')}</select></div>
    <div class="form-group"><label>Size (MB)</label><input id="f-vol-size" type="number" value="1024" min="128"></div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Cancel</button>
      <button class="btn btn-primary" onclick="createVolume()">Create</button>
    </div>
  `);
}
async function createVolume() {
  const tenant_id = document.getElementById('f-vol-tenant').value;
  const size_mb = parseInt(document.getElementById('f-vol-size').value);
  await api('POST','/volumes',{tenant_id,size_mb});
  closeModal(); renderVolumes();
}

// ─── Workspaces ─────────────────────────────────────────────────
async function renderWorkspaces() {
  await loadAll();
  document.getElementById('page-workspaces').innerHTML = `
    <div class="toolbar"><div><h1>Workspaces</h1><p class="subtitle">Compute workspaces with attached storage</p></div>
      <button class="btn btn-primary" onclick="showCreateWorkspace()">+ Launch Workspace</button></div>
    ${workspaces.length
      ? renderTable(['Workspace','Tenant','Image','Status','SSH','Actions'], workspaces.map(w=>[
          `<a href="#" onclick="event.preventDefault();showWorkspace('${w.id}')">${short(w.id)}</a>`,
          short(w.tenant_id), w.image,
          `<span class="badge badge-${w.status}">${w.status}</span>`,
          w.status==='running' ? `<code>localhost:${w.port}</code>` : '—',
          w.status==='running'
            ? `<button class="btn btn-sm btn-danger" onclick="deleteWorkspace('${w.id}')">Terminate</button>`
            : ''
        ]))
      : '<div class="empty"><p>No workspaces. Launch one to get started!</p></div>'}
  `;
}

function showCreateWorkspace() {
  if (!tenants.length) return alert('Create a tenant first');
  showModal('Launch Workspace', `
    <div class="form-group"><label>Tenant</label>
      <select id="f-ws-tenant">${tenants.map(t=>`<option value="${t.id}">${esc(t.name)}</option>`).join('')}</select></div>
    <div class="form-group"><label>Image</label>
      <input id="f-ws-image" value="alpine:latest" placeholder="e.g. ubuntu:22.04"></div>
    <div class="form-group"><label>Volume Size (MB)</label>
      <input id="f-ws-size" type="number" value="512" min="128"></div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Cancel</button>
      <button class="btn btn-primary" onclick="createWorkspace()">Launch</button>
    </div>
  `);
}
async function createWorkspace() {
  const tenant_id = document.getElementById('f-ws-tenant').value;
  const image = document.getElementById('f-ws-image').value.trim();
  const size_mb = parseInt(document.getElementById('f-ws-size').value);
  await api('POST','/workspaces',{tenant_id,image,size_mb});
  closeModal(); renderWorkspaces();
}

async function deleteWorkspace(id) {
  if (!confirm('Terminate this workspace? All data on the volume will be lost.')) return;
  await api('DELETE',`/workspaces/${id}`);
  renderWorkspaces();
}

async function showWorkspace(id) {
  let w;
  try { w = await api('GET',`/workspaces/${id}`); } catch(e) { return alert('Workspace not found'); }
  showModal('Workspace Details', `
    <div class="detail-grid">
      <div class="detail-item"><div class="label">ID</div><div class="val">${w.id}</div></div>
      <div class="detail-item"><div class="label">Status</div><div class="val"><span class="badge badge-${w.status}">${w.status}</span></div></div>
      <div class="detail-item"><div class="label">Image</div><div class="val">${w.image}</div></div>
      <div class="detail-item"><div class="label">Tenant</div><div class="val">${short(w.tenant_id)}</div></div>
      <div class="detail-item"><div class="label">Volume ID</div><div class="val">${w.volume_id||'—'}</div></div>
      <div class="detail-item"><div class="label">Container ID</div><div class="val">${short(w.container_id)||'—'}</div></div>
      ${w.status==='running' ? `<div class="detail-item full"><div class="label">SSH Access</div>
        <div class="ssh-box"><code>ssh root@localhost -p ${w.port}</code>
        <span class="copy" onclick="copySSH(${w.port})">📋 Copy</span></div></div>` : ''}
    </div>
    <div class="form-actions">
      <button class="btn" onclick="closeModal()">Close</button>
      ${w.status==='running' ? `<button class="btn btn-danger" onclick="closeModal();deleteWorkspace('${w.id}')">Terminate</button>` : ''}
    </div>
  `);
}

function copySSH(port) {
  navigator.clipboard.writeText(`ssh root@localhost -p ${port}`);
}

// ─── Helpers ────────────────────────────────────────────────────
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

// ─── Init ───────────────────────────────────────────────────────
renderPage('dashboard');
