interface Props {
  value: number
  goal: number
  size?: number
  stroke?: number
  label?: string
  sublabel?: string
}

// GoalRing renders a circular progress indicator toward a daily code-line goal.
// It is the visual anchor of the dashboard hero, making "did I hit my goal
// today?" the first thing the user sees.
export default function GoalRing({ value, goal, size = 96, stroke = 9, label, sublabel }: Props) {
  const radius = (size - stroke) / 2
  const circumference = 2 * Math.PI * radius
  const ratio = goal > 0 ? Math.min(value / goal, 1) : 0
  const offset = circumference * (1 - ratio)
  const pct = Math.round(ratio * 100)
  const reached = value >= goal && goal > 0

  const color = reached ? 'var(--accent)' : ratio >= 0.5 ? 'var(--accent-light)' : 'var(--warning)'

  return (
    <div className="goal-ring" style={{ width: size, height: size }}>
      <svg width={size} height={size} className="goal-ring-svg">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="var(--bg-tertiary)"
          strokeWidth={stroke}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
          className="goal-ring-progress"
        />
      </svg>
      <div className="goal-ring-center">
        <span className="goal-ring-value">{pct}%</span>
        {label && <span className="goal-ring-label">{label}</span>}
      </div>
      {sublabel && <span className="goal-ring-sub">{sublabel}</span>}
    </div>
  )
}
