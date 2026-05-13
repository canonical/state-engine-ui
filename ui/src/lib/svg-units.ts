export function parseSvgLength(value: string): number {
  const match = value.match(/^([0-9.]+)\s*(in|pt|px|cm|mm)?$/)
  if (!match) return 0
  const n = parseFloat(match[1])
  const unit = match[2] ?? 'pt'
  switch (unit) {
    case 'in':
      return n * 96
    case 'pt':
      return n * (96 / 72)
    case 'px':
      return n
    case 'cm':
      return n * (96 / 2.54)
    case 'mm':
      return n * (96 / 25.4)
    default:
      return n
  }
}
