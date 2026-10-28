const http = require('http'), fs = require('fs'), path = require('path');
const env = {};
try { fs.readFileSync(path.join(__dirname, '.env'), 'utf8').split(/\r?\n/).forEach(l => { const m = l.match(/^\s*([A-Z_]+)\s*=\s*(.*)$/); if (m) env[m[1]] = m[2]; }); } catch {}
const E = k => process.env[k] || env[k] || '';
const API = E('SUPABASE_URL').replace(/\/$/, '') + '/functions/v1/bm-licensing';
const { packageName } = require('./apk');
const sb = async (op, data) => { const r = await fetch(API + '/' + op, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + E('ADMIN_TOKEN') }, body: JSON.stringify(data || {}) }); const j = await r.json(); if (!j.ok) throw new Error(j.error || 'error'); return j; };
const body = req => new Promise(res => { const c = []; req.on('data', d => c.push(d)); req.on('end', () => res(Buffer.concat(c))); });
const send = (res, code, data, type = 'application/json', extra = {}) => { res.writeHead(code, { 'Content-Type': type, ...extra }); res.end(Buffer.isBuffer(data) || typeof data === 'string' ? data : JSON.stringify(data)); };

function buildExe(apk, meta) {
  const stub = fs.readFileSync(path.join(__dirname, 'stub', 'launcher.exe'));
  const m = Buffer.from(JSON.stringify(meta)), foot = Buffer.alloc(24);
  foot.writeBigUInt64LE(BigInt(m.length), 0); foot.writeBigUInt64LE(BigInt(apk.length), 8); foot.write('APKBOX01', 16, 'ascii');
  return Buffer.concat([stub, apk, m, foot]);
}

http.createServer(async (req, res) => {
  try {
    const u = new URL(req.url, 'http://x');
    if (u.pathname === '/') return send(res, 200, fs.readFileSync(path.join(__dirname, 'public', 'index.html')), 'text/html; charset=utf-8');
    if (req.headers['x-admin-password'] !== E('ADMIN_PASSWORD')) return send(res, 401, { error: 'unauthorized' });
    if (u.pathname === '/api/licenses') return send(res, 200, await sb('admin-list'));
    if (u.pathname === '/api/action' && req.method === 'POST') return send(res, 200, await sb('admin-action', JSON.parse(await body(req))));
    if (u.pathname === '/api/export' && req.method === 'POST') {
      const name = (u.searchParams.get('name') || 'App').replace(/[^\w\u0600-\u06FF -]/g, '').trim() || 'App';
      const days = +u.searchParams.get('days') || 30;
      const apk = await body(req);
      let pkg; try { pkg = packageName(apk); } catch { return send(res, 400, { error: 'not a valid APK' }); }
      const { license: lic } = await sb('admin-create', { subscriber: u.searchParams.get('customer') || '-', app_name: name, app_package: pkg, duration_days: days, self_destruct: u.searchParams.get('self_destruct') === '1' });
      const exe = buildExe(apk, { app_name: name, app_package: pkg, license_key: lic.license_key, api_base: API, runtime_url: E('RUNTIME_URL'), runtime_sha256: E('RUNTIME_SHA256') });
      return send(res, 200, exe, 'application/octet-stream', { 'Content-Disposition': `attachment; filename*=UTF-8''${encodeURIComponent(name)}.exe` });
    }
    send(res, 404, { error: 'not found' });
  } catch (e) { send(res, 500, { error: String(e.message || e) }); }
}).listen(+E('PORT') || 8080, () => console.log('Admin panel: http://localhost:' + (E('PORT') || 8080)));
