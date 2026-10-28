// Extract package name from an APK's binary AndroidManifest.xml (no dependencies).
const zlib = require('zlib');
function manifest(apk) {
  let e = apk.length - 22; while (e >= 0 && apk.readUInt32LE(e) !== 0x06054b50) e--;
  if (e < 0) throw Error('not a zip');
  let p = apk.readUInt32LE(e + 16); const n = apk.readUInt16LE(e + 10);
  for (let i = 0; i < n; i++) {
    const method = apk.readUInt16LE(p + 10), csize = apk.readUInt32LE(p + 20), nl = apk.readUInt16LE(p + 28), xl = apk.readUInt16LE(p + 30), cl = apk.readUInt16LE(p + 32), lh = apk.readUInt32LE(p + 42);
    const name = apk.toString('utf8', p + 46, p + 46 + nl);
    if (name === 'AndroidManifest.xml') {
      const d = lh + 30 + apk.readUInt16LE(lh + 26) + apk.readUInt16LE(lh + 28), raw = apk.subarray(d, d + csize);
      return method === 0 ? raw : zlib.inflateRawSync(raw);
    }
    p += 46 + nl + xl + cl;
  }
  throw Error('no manifest');
}
function packageName(apk) {
  const b = manifest(apk); let strings = [];
  for (let o = 8; o + 8 <= b.length;) {
    const type = b.readUInt16LE(o), size = b.readUInt32LE(o + 4);
    if (type === 0x0001) { // string pool
      const cnt = b.readUInt32LE(o + 8), flags = b.readUInt32LE(o + 16), sStart = b.readUInt32LE(o + 20), utf8 = flags & 0x100;
      for (let i = 0; i < cnt; i++) {
        let q = o + sStart + b.readUInt32LE(o + 28 + i * 4);
        if (utf8) { let l = b[q++]; if (l & 0x80) q++; let bl = b[q++]; if (bl & 0x80) bl = ((bl & 0x7f) << 8) | b[q++]; strings.push(b.toString('utf8', q, q + bl)); }
        else { let l = b.readUInt16LE(q); q += 2; if (l & 0x8000) { l = ((l & 0x7fff) << 16) | b.readUInt16LE(q); q += 2; } strings.push(b.toString('utf16le', q, q + l * 2)); }
      }
    } else if (type === 0x0102) { // start element: first one is <manifest>
      const ac = b.readUInt16LE(o + 28), as = o + 16 + b.readUInt16LE(o + 24);
      for (let i = 0; i < ac; i++) { const a = as + i * 20; if (strings[b.readUInt32LE(a + 4)] === 'package') return strings[b.readUInt32LE(a + 8)]; }
      break;
    }
    o += size;
  }
  throw Error('package not found');
}
module.exports = { packageName };
