import fs from 'node:fs'
import path from 'node:path'
import sharp from 'sharp'

const SIZES = [512, 256, 128, 64, 48, 32, 16]

async function main() {
  const svgPath = path.resolve(process.cwd(), 'build/icon.svg')
  const outDir = path.resolve(process.cwd(), 'build/icons')

  if (!fs.existsSync(svgPath)) {
    console.error(`SVG not found: ${svgPath}`)
    process.exit(1)
  }

  fs.mkdirSync(outDir, { recursive: true })

  const svgBuffer = fs.readFileSync(svgPath)

  for (const size of SIZES) {
    const outPath = path.join(outDir, `icon-${size}.png`)
    await sharp(svgBuffer)
      .resize(size, size)
      .png()
      .toFile(outPath)
    console.log(`  ✓ icon-${size}.png`)
  }

  // Copy 512 as appicon.png for Wails
  const appIconPath = path.resolve(process.cwd(), 'build/appicon.png')
  fs.copyFileSync(path.join(outDir, 'icon-512.png'), appIconPath)
  console.log(`  ✓ appicon.png (512x512 for Wails)`)

  // Generate .ico for Windows
  const icoPath = path.join(outDir, 'icon.ico')
  const icoSizes = [256, 128, 64, 48, 32, 16]
  const pngBuffers = []
  for (const size of icoSizes) {
    const buf = await sharp(svgBuffer).resize(size, size).png().toBuffer()
    pngBuffers.push({ size, buf })
  }

  // Build ICO file manually
  // ICO header: 6 bytes (reserved=0, type=1, count=n)
  // Then n entries of 16 bytes each (width, height, colors, reserved, planes, bpp, size, offset)
  // Then the PNG data
  const headerSize = 6 + icoSizes.length * 16
  let dataOffset = headerSize
  let totalSize = headerSize

  const entries = []
  for (const { size, buf } of pngBuffers) {
    totalSize += buf.length
    entries.push({
      width: size >= 256 ? 0 : size,
      height: size >= 256 ? 0 : size,
      colors: 0,
      reserved: 0,
      planes: 1,
      bpp: 32,
      size: buf.length,
      offset: dataOffset,
      buf,
    })
    dataOffset += buf.length
  }

  const icoBuffer = Buffer.alloc(totalSize)
  // Header
  icoBuffer.writeUInt16LE(0, 0) // reserved
  icoBuffer.writeUInt16LE(1, 2) // type: ICO
  icoBuffer.writeUInt16LE(icoSizes.length, 4) // count

  // Entries
  let entryOffset = 6
  for (const entry of entries) {
    icoBuffer.writeUInt8(entry.width, entryOffset)
    icoBuffer.writeUInt8(entry.height, entryOffset + 1)
    icoBuffer.writeUInt8(entry.colors, entryOffset + 2)
    icoBuffer.writeUInt8(entry.reserved, entryOffset + 3)
    icoBuffer.writeUInt16LE(entry.planes, entryOffset + 4)
    icoBuffer.writeUInt16LE(entry.bpp, entryOffset + 6)
    icoBuffer.writeUInt32LE(entry.size, entryOffset + 8)
    icoBuffer.writeUInt32LE(entry.offset, entryOffset + 12)
    entryOffset += 16
  }

  // PNG data
  for (const entry of entries) {
    entry.buf.copy(icoBuffer, entry.offset)
  }

  fs.writeFileSync(icoPath, icoBuffer)
  console.log(`  ✓ icon.ico (${icoSizes.length} sizes)`)

  console.log(`\nAll icons generated in ${outDir}`)
}

main().catch((err) => {
  console.error('Failed to generate icons:', err)
  process.exit(1)
})
