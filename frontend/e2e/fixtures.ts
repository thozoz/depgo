import zlib from 'zlib';

/**
 * Calculates standard CRC-32 for PNG chunk integrity.
 */
function crc32(buf: Buffer): number {
  let c = ~0;
  for (let i = 0; i < buf.length; i++) {
    c ^= buf[i];
    for (let j = 0; j < 8; j++) {
      c = (c >>> 1) ^ (0xedb88320 & -(c & 1));
    }
  }
  return ~c >>> 0;
}

function makePngChunk(type: string, data: Buffer): Buffer {
  const typeBuf = Buffer.from(type, 'ascii');
  const lenBuf = Buffer.alloc(4);
  lenBuf.writeUInt32BE(data.length, 0);

  const crcBuf = Buffer.alloc(4);
  const typeAndData = Buffer.concat([typeBuf, data]);
  crcBuf.writeUInt32BE(crc32(typeAndData), 0);

  return Buffer.concat([lenBuf, typeBuf, data, crcBuf]);
}

/**
 * Creates a representative, visible 400x240 RGB PNG test image:
 * Charcoal background (#18181B) with an emerald accent border and central banner.
 */
export function generateVisibleTestPng(width = 400, height = 240): Buffer {
  const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

  // IHDR: 13 bytes
  const ihdrData = Buffer.alloc(13);
  ihdrData.writeUInt32BE(width, 0);
  ihdrData.writeUInt32BE(height, 4);
  ihdrData[8] = 8; // 8-bit depth
  ihdrData[9] = 2; // RGB (3 bytes/pixel)
  ihdrData[10] = 0; // compression
  ihdrData[11] = 0; // filter
  ihdrData[12] = 0; // interlace
  const ihdr = makePngChunk('IHDR', ihdrData);

  // Scanlines: each row has 1 filter byte (0) + width * 3 bytes (R, G, B)
  const rowBytes = 1 + width * 3;
  const rawData = Buffer.alloc(rowBytes * height);

  for (let y = 0; y < height; y++) {
    const rowOffset = y * rowBytes;
    rawData[rowOffset] = 0; // filter None

    const isBorder = y < 8 || y >= height - 8;
    const isCenterStripe = y >= 90 && y <= 150;

    for (let x = 0; x < width; x++) {
      const pxOffset = rowOffset + 1 + x * 3;
      const isColBorder = x < 8 || x >= width - 8;

      if (isBorder || isColBorder) {
        // Emerald border (#10B981)
        rawData[pxOffset] = 0x10;
        rawData[pxOffset + 1] = 0xb9;
        rawData[pxOffset + 2] = 0x81;
      } else if (isCenterStripe && x >= 30 && x <= width - 30) {
        // Bright emerald banner (#34D399)
        rawData[pxOffset] = 0x34;
        rawData[pxOffset + 1] = 0xd3;
        rawData[pxOffset + 2] = 0x99;
      } else {
        // Charcoal canvas (#18181B)
        rawData[pxOffset] = 0x18;
        rawData[pxOffset + 1] = 0x18;
        rawData[pxOffset + 2] = 0x1b;
      }
    }
  }

  const compressed = zlib.deflateSync(rawData);
  const idat = makePngChunk('IDAT', compressed);
  const iend = makePngChunk('IEND', Buffer.alloc(0));

  return Buffer.concat([sig, ihdr, idat, iend]);
}

/**
 * Creates a spec-compliant PDF with Helvetica font and visible text "Depgo PDF preview test".
 * Computes exact byte offsets for xref table so any PDF reader parses it cleanly.
 */
export function generateVisibleTestPdf(): Buffer {
  const streamText = `BT
/F1 22 Tf
40 180 Td
(Depgo PDF preview test) Tj
0 -30 Td
/F1 14 Tf
(Automated E2E document inspection) Tj
0 -25 Td
(Yerel tarayici onizleme dogrulamasi) Tj
ET`;
  const streamLength = Buffer.byteLength(streamText, 'utf-8');

  // Build objects
  const header = '%PDF-1.4\n';

  const obj1 = '1 0 obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\n';
  const obj2 = '2 0 obj\n<</Type /Pages /Count 1 /Kids [3 0 R]>>\nendobj\n';
  const obj3 =
    '3 0 obj\n<</Type /Page /Parent 2 0 R /MediaBox [0 0 450 250] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R>>\nendobj\n';
  const obj4 = '4 0 obj\n<</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>\nendobj\n';
  const obj5 = `5 0 obj\n<</Length ${streamLength}>>\nstream\n${streamText}\nendstream\nendobj\n`;

  // Compute exact offsets
  let offset = Buffer.byteLength(header, 'utf-8');
  const offset1 = offset;
  offset += Buffer.byteLength(obj1, 'utf-8');
  const offset2 = offset;
  offset += Buffer.byteLength(obj2, 'utf-8');
  const offset3 = offset;
  offset += Buffer.byteLength(obj3, 'utf-8');
  const offset4 = offset;
  offset += Buffer.byteLength(obj4, 'utf-8');
  const offset5 = offset;
  offset += Buffer.byteLength(obj5, 'utf-8');

  const xrefOffset = offset;
  const pad = (n: number) => n.toString().padStart(10, '0');

  const xref = `xref
0 6
0000000000 65535 f
${pad(offset1)} 00000 n
${pad(offset2)} 00000 n
${pad(offset3)} 00000 n
${pad(offset4)} 00000 n
${pad(offset5)} 00000 n
trailer
<</Size 6 /Root 1 0 R>>
startxref
${xrefOffset}
%%EOF
`;

  return Buffer.from(header + obj1 + obj2 + obj3 + obj4 + obj5 + xref, 'utf-8');
}

export const VISIBLE_PNG_BUFFER = generateVisibleTestPng(400, 240);
export const VISIBLE_PDF_BUFFER = generateVisibleTestPdf();

export const SAMPLE_TEXT_CONTENT = `# Depgo Test Dosyası
Bu dosya otomatik E2E testi tarafından oluşturulmuştur.
Türkçe karakterler: ç, ğ, ı, ö, ş, ü, İ.`;

export const SAMPLE_TEXT_BUFFER = Buffer.from(SAMPLE_TEXT_CONTENT, 'utf-8');
