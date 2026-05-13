import { useState } from 'react'
import { ChangeGraph } from './components/ChangeGraph'
import TaskSidebar from './components/TaskSidebar'
import { sampleChange } from './data/sampleChange'

function App() {
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)

  const selectedTask =
    sampleChange.tasks.find((t) => t.id === selectedTaskId) ?? null

  return (
    <div className="flex h-screen w-full overflow-hidden bg-zinc-900">
      <div className="w-[65%] p-6">
        <ChangeGraph
          change={sampleChange}
          selectedTaskId={selectedTaskId}
          onSelectTask={setSelectedTaskId}
        />
      </div>
      <div className="w-[35%] border-l border-zinc-700 bg-white">
        <TaskSidebar
          tasks={sampleChange.tasks}
          selectedTaskId={selectedTaskId}
          selectedTask={selectedTask}
          onClearSelection={() => setSelectedTaskId(null)}
        />
      </div>
    </div>
  )
}

export default App
