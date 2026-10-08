// Re-encode the source PNGs into small, browser-friendly assets.
// Pure Node: decodes the PNG (RGBA, filter types 0-4), box-downsamples it and re-encodes with zlib.
const fs = require('fs');
const zlib = require('zlib');

function decodePNG(buf) {
  if (buf.readUInt32BE(0) !== 0x89504e47) throw new Error('not a png');
  let o = 8, w = 0, h = 0, bd = 0, ct = 0, idat = [];
  while (o < buf.length) {
    const len = buf.readUInt32BE(o);
    const type = buf.slice(o + 4, o + 8).toString('latin1');
    const data = buf.slice(o + 8, o + 8 + len);
    if (type === 'IHDR') { w = data.readUInt32BE(0); h = data.readUInt32BE(4); bd = data[8]; ct = data[9]; if (data[12] !== 0) throw new Error('interlaced not supported'); }
    else if (type === 'IDAT') idat.push(data);
    else if (type === 'IEND') break;
    o += 12 + len;
  }
  if (bd !== 8 || ct !== 6) throw new Error('expected 8-bit RGBA, got bd=' + bd + ' ct=' + ct);
  const raw = zlib.inflateSync(Buffer.concat(idat));
  const bpp = 4, stride = w * bpp;
  const out = Buffer.alloc(h * stride);
  let pos = 0;
  for (let y = 0; y < h; y++) {
    const ft = raw[pos++];
    const line = raw.slice(pos, pos + stride); pos += stride;
    const cur = out.slice(y * stride, (y + 1) * stride);
    const prev = y > 0 ? out.slice((y - 1) * stride, y * stride) : null;
    for (let x = 0; x < stride; x++) {
      const a = x >= bpp ? cur[x - bpp] : 0;
      const b = prev ? prev[x] : 0;
      const c = prev && x >= bpp ? prev[x - bpp] : 0;
      const v = line[x];
      let r;
      switch (ft) {
        case 0: r = v; break;
        case 1: r = v + a; break;
        case 2: r = v + b; break;
        case 3: r = v + ((a + b) >> 1); break;
        case 4: { const p = a + b - c, pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c); r = v + (pa <= pb && pa <= pc ? a : pb <= pc ? b : c); break; }
        default: throw new Error('bad filter ' + ft);
      }
      cur[x] = r & 0xff;
    }
  }
  return { w, h, data: out };
}

// Box-filter downscale (average of the source block). Good quality for logos.
function resize(img, nw, nh) {
  const { w, h, data } = img;
  const out = Buffer.alloc(nw * nh * 4);
  const xr = w / nw, yr = h / nh;
  for (let y = 0; y < nh; y++) {
    const y0 = Math.floor(y * yr), y1 = Math.max(y0 + 1, Math.floor((y + 1) * yr));
    for (let x = 0; x < nw; x++) {
      const x0 = Math.floor(x * xr), x1 = Math.max(x0 + 1, Math.floor((x + 1) * xr));
      let r = 0, g = 0, b = 0, a = 0, n = 0;
      for (let sy = y0; sy < y1; sy++) for (let sx = x0; sx < x1; sx++) {
        const i = (sy * w + sx) * 4;
        const al = data[i + 3] / 255;
        r += data[i] * al; g += data[i + 1] * al; b += data[i + 2] * al; a += data[i + 3]; n++;
      }
      const o = (y * nw + x) * 4;
      const wa = a / n / 255;
      out[o] = wa ? Math.round(r / n / wa) : 0;
      out[o + 1] = wa ? Math.round(g / n / wa) : 0;
      out[o + 2] = wa ? Math.round(b / n / wa) : 0;
      out[o + 3] = Math.round(a / n);
    }
  }
  return { w: nw, h: nh, data: out };
}

function encodePNG(img) {
  const { w, h, data } = img;
  const stride = w * 4;
  const raw = Buffer.alloc(h * (stride + 1));
  for (let y = 0; y < h; y++) {
    raw[y * (stride + 1)] = 0;
    data.copy(raw, y * (stride + 1) + 1, y * stride, (y + 1) * stride);
  }
  const parts = [];
  const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  const chunk = (type, body) => {
    const len = Buffer.alloc(4); len.writeUInt32BE(body.length);
    const t = Buffer.from(type, 'latin1');
    const crcTable = [];
    for (let n = 0; n < 256; n++) { let c = n; for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1; crcTable[n] = c >>> 0; }
    let c = 0xffffffff;
    for (const byte of Buffer.concat([t, body])) c = crcTable[(c ^ byte) & 0xff] ^ (c >>> 8);
    const crc = Buffer.alloc(4); crc.writeUInt32BE((c ^ 0xffffffff) >>> 0);
    return Buffer.concat([len, t, body, crc]);
  };
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(w, 0); ihdr.writeUInt32BE(h, 4); ihdr[8] = 8; ihdr[9] = 6;
  return Buffer.concat([sig, chunk('IHDR', ihdr), chunk('IDAT', zlib.deflateSync(raw, { level: 9 })), chunk('IEND', Buffer.alloc(0))]);
}

const src = decodePNG(fs.readFileSync('../favicon.png'));
const logo = decodePNG(fs.readFileSync('../logo.png'));
console.log('source favicon', src.w + 'x' + src.h, 'logo', logo.w + 'x' + logo.h);

const jobs = [
  // square icon: 512 / 192 / 180 / 64 / 32
  [src, 512, 512, 'assets/favicon-512.png'],
  [src, 192, 192, 'assets/favicon-192.png'],
  [src, 180, 180, 'assets/apple-touch-icon.png'],
  [src, 64, 64, 'assets/favicon-64.png'],
  [src, 32, 32, 'assets/favicon-32.png'],
  // wide lockup: keep the aspect ratio 2143x734
  [logo, 1080, 370, 'assets/logo-1080.png'],
  [logo, 540, 185, 'assets/logo-540.png'],
  [logo, 360, 123, 'assets/logo-360.png'],
];
fs.mkdirSync('assets', { recursive: true });
for (const [im, w, h, out] of jobs) {
  const png = encodePNG(resize(im, w, h));
  fs.writeFileSync(out, png);
  console.log(out, w + 'x' + h, Math.round(png.length / 1024) + 'KB');
}
