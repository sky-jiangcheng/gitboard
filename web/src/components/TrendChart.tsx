import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js'
import { Line } from 'react-chartjs-2'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend)

export interface TrendDataset {
  label: string
  data: number[]
  color: string
}

interface Props {
  labels: string[]
  datasets: TrendDataset[]
}

function TrendChart({ labels, datasets }: Props) {
  if (labels.length === 0) {
    return <div className="chart-empty">暂无趋势数据</div>
  }

  const data = {
    labels,
    datasets: datasets.map((ds) => ({
      label: ds.label,
      data: ds.data,
      borderColor: ds.color,
      backgroundColor: ds.color + '18',
      fill: true,
      tension: 0.3,
      pointRadius: 3,
      pointHoverRadius: 5,
    })),
  }

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index' as const,
      intersect: false,
    },
    plugins: {
      legend: { display: true, position: 'top' as const, labels: { usePointStyle: true, padding: 20 } },
      tooltip: { backgroundColor: '#1a1a2e', titleColor: '#fff', bodyColor: '#ccc' },
    },
    scales: {
      y: {
        beginAtZero: true,
        title: { display: true, text: '数量', color: '#999' },
        grid: { color: '#e8e8e8' },
      },
      x: {
        title: { display: true, text: '日期', color: '#999' },
        grid: { display: false },
      },
    },
  }

  return (
    <div className="chart-container">
      <Line data={data} options={options} />
    </div>
  )
}

export default TrendChart
