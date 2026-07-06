interface Props {
  value: string
  onChange: (date: string) => void
}

function getToday(): string {
  return new Date().toISOString().split('T')[0]
}

function getYesterday(): string {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return d.toISOString().split('T')[0]
}

function DatePicker({ value, onChange }: Props) {
  const today = getToday()
  const yesterday = getYesterday()

  return (
    <div className="date-picker">
      <button
        className={`btn btn-sm ${value === yesterday ? 'btn-active' : ''}`}
        onClick={() => onChange(yesterday)}
      >
        昨天
      </button>
      <button
        className={`btn btn-sm ${value === today ? 'btn-active' : ''}`}
        onClick={() => onChange(today)}
      >
        今天
      </button>
      <input
        type="date"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="form-input date-input"
      />
    </div>
  )
}

export default DatePicker
