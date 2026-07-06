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

interface Props {
  labels: string[]
  values: number[]
}

function TrendChart({ labels, values }: Props) {
  if (labels.length === 0) {
    return <div className="chart-empty">暂无趋势数据</div>
  }

  const data = {
    labels,
    datasets: [
      {
        label: '新增行数',
        data: values,
        borderColor: '#4caf50',
        backgroundColor: 'rgba(76, 175, 80, 0.1)',
        fill: true,
        tension: 0.3,
      },
    ],
  }

  const options = {
    responsive: true,
    plugins: {
      legend: { display: true, position: 'top' as const },
    },
    scales: {
      y: {
        beginAtZero: true,
        title: { display: true, text: '行数' },
      },
      x: {
        title: { display: true, text: '日期' },
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
