// cognitive.js — Cognitive Graph (CGM) tab. D3 force-directed "balls connected"
// view of an agent's per-agent twin graph (roadmap_opus8.md §4.9, D14).
// Reads /api/agents/cognitive/graph?id=<agent> + /api/agents/cognitive/tensions.
// Reuses the vendored D3 (pattern from codemap.js).

const TYPE_COLOR = {
  person: '#f472b6', preference: '#a78bfa', trait: '#c084fc', concept: '#38bdf8',
  project: '#34d399', event: '#fbbf24', skill: '#4ade80', fact: '#67e8f9',
  knowledge: '#22d3ee', doctrine: '#fb7185', persona: '#818cf8', memory: '#fcd34d',
};
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

function loadD3() {
  return new Promise((resolve, reject) => {
    if (window.d3) return resolve(window.d3);
    const s = document.createElement('script');
    s.src = '/vendor/d3.min.js';
    s.onload = () => (window.d3 ? resolve(window.d3) : reject(new Error('d3 missing')));
    s.onerror = () => reject(new Error('failed to load /vendor/d3.min.js'));
    document.head.appendChild(s);
  });
}

export async function render(container) {
  container.innerHTML = `
    <div style="padding:14px 18px">
      <div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-bottom:10px">
        <h2 style="margin:0;color:#c4b5fd;font-weight:600">🧠 Cognitive Graph</h2>
        <input id="cgAgent" value="mr-flow" spellcheck="false"
          style="background:#0b1020;border:1px solid #334155;color:#e2e8f0;border-radius:6px;padding:5px 10px;font-size:.82rem;width:160px"/>
        <button id="cgLoad" class="btn-primary" style="padding:5px 14px;font-size:.8rem">Load</button>
        <span id="cgStats" style="font-size:.78rem;color:#94a3b8"></span>
      </div>
      <div style="display:flex;gap:12px;flex-wrap:wrap;font-size:.72rem;color:#94a3b8;margin-bottom:8px">
        <span>● <span style="color:#f472b6">person</span></span>
        <span>● <span style="color:#a78bfa">preference/trait</span></span>
        <span>● <span style="color:#34d399">project</span></span>
        <span>● <span style="color:#4ade80">skill</span></span>
        <span>● <span style="color:#fbbf24">event</span></span>
        <span>dashed = shadow (not yet promoted)</span>
      </div>
      <svg id="cgSvg" width="100%" height="560"
        style="background:radial-gradient(circle at 50% 40%, #0e1530, #070a16);border:1px solid #1e293b;border-radius:10px"></svg>
      <div id="cgTensions" style="margin-top:12px;font-size:.8rem"></div>
    </div>`;

  const d3 = await loadD3().catch((e) => { container.querySelector('#cgSvg').outerHTML = `<div class="err">❌ ${esc(e.message)}</div>`; return null; });
  if (!d3) return;

  const load = () => draw(d3, container, container.querySelector('#cgAgent').value.trim() || 'mr-flow');
  container.querySelector('#cgLoad').onclick = load;
  load();
}

async function draw(d3, container, agentId) {
  const svgEl = container.querySelector('#cgSvg');
  const stats = container.querySelector('#cgStats');
  stats.textContent = 'loading…';
  let data;
  try {
    data = await (await fetch(`/api/agents/cognitive/graph?id=${encodeURIComponent(agentId)}`)).json();
  } catch (e) { stats.textContent = 'error: ' + e.message; return; }
  if (data.error) { stats.textContent = 'error: ' + data.error; return; }

  const nodes = (data.nodes || []).map((n) => ({ ...n }));
  const idset = new Set(nodes.map((n) => n.id));
  const links = (data.edges || []).filter((e) => idset.has(e.from_id) && idset.has(e.to_id))
    .map((e) => ({ source: e.from_id, target: e.to_id, rel: e.relation_type, status: e.status }));
  stats.textContent = `${nodes.length} nodes · ${links.length} edges · agent: ${agentId}`;

  const W = svgEl.clientWidth || 900, H = 560;
  const svg = d3.select(svgEl); svg.selectAll('*').remove();
  const root = svg.append('g');
  svg.call(d3.zoom().scaleExtent([0.2, 4]).on('zoom', (ev) => root.attr('transform', ev.transform)));

  const radius = (n) => 9 + Math.min(14, (n.hit_count || 1) * 1.6);

  const link = root.append('g').attr('stroke', '#475569').attr('stroke-opacity', 0.6)
    .selectAll('line').data(links).join('line')
    .attr('stroke-width', 1.4).attr('stroke-dasharray', (d) => d.status === 'shadow' ? '4 3' : null);

  const linkLabel = root.append('g').selectAll('text').data(links).join('text')
    .text((d) => d.rel).attr('font-size', 8).attr('fill', '#64748b').attr('text-anchor', 'middle');

  const node = root.append('g').selectAll('g').data(nodes).join('g').call(drag(d3));
  node.append('circle').attr('r', radius)
    .attr('fill', (n) => TYPE_COLOR[n.type] || '#94a3b8')
    .attr('opacity', (n) => n.status === 'shadow' ? 0.45 : 0.95)
    .attr('stroke', (n) => n.status === 'quarantined' ? '#ef4444' : '#0b1020').attr('stroke-width', 2);
  node.append('title').text((n) => `${n.label} (${n.type}) · conf ${n.confidence} · ${n.status} · hits ${n.hit_count}`);
  node.append('text').text((n) => n.label).attr('x', (n) => radius(n) + 4).attr('y', 4)
    .attr('font-size', 11).attr('fill', '#cbd5e1');

  const sim = d3.forceSimulation(nodes)
    .force('link', d3.forceLink(links).id((d) => d.id).distance(120).strength(0.3))
    .force('charge', d3.forceManyBody().strength(-340).distanceMax(420))
    .force('collide', d3.forceCollide((d) => radius(d) + 18))
    .force('center', d3.forceCenter(W / 2, H / 2).strength(0.06))
    .on('tick', () => {
      link.attr('x1', (d) => d.source.x).attr('y1', (d) => d.source.y).attr('x2', (d) => d.target.x).attr('y2', (d) => d.target.y);
      linkLabel.attr('x', (d) => (d.source.x + d.target.x) / 2).attr('y', (d) => (d.source.y + d.target.y) / 2);
      node.attr('transform', (d) => `translate(${d.x},${d.y})`);
    });

  drawTensions(container, agentId);
  function drag(d3) {
    return d3.drag()
      .on('start', (ev, d) => { if (!ev.active) sim.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y; })
      .on('drag', (ev, d) => { d.fx = ev.x; d.fy = ev.y; })
      .on('end', (ev, d) => { if (!ev.active) sim.alphaTarget(0); d.fx = null; d.fy = null; });
  }
}

async function drawTensions(container, agentId) {
  const el = container.querySelector('#cgTensions');
  let data;
  try { data = await (await fetch(`/api/agents/cognitive/tensions?id=${encodeURIComponent(agentId)}`)).json(); } catch { return; }
  const items = data.items || [];
  if (!items.length) { el.innerHTML = `<span style="color:#475569">No open contradictions.</span>`; return; }
  el.innerHTML = `<div style="color:#fbbf24;font-weight:600;margin-bottom:4px">⚠ Open contradictions (owner decides)</div>` +
    items.map((t) => `<div style="color:#cbd5e1">• ${esc(t.from_id)} —${esc(t.relation_type)}→ <s>${esc(t.old_to_id)}</s> / ${esc(t.new_to_id)}</div>`).join('');
}
