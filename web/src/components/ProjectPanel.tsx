import TodoSection from './TodoSection'
import NoteSection from './NoteSection'

interface Props {
  projectId: number
}

function ProjectPanel({ projectId }: Props) {
  return (
    <div className="side-panel">
      <TodoSection projectId={projectId} />
      <NoteSection projectId={projectId} />
    </div>
  )
}

export default ProjectPanel
