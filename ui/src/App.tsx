import { ChangeGraph } from './components/ChangeGraph'
import TaskSidebar from './components/TaskSidebar'
import { sampleChange } from './data/sampleChange'

function App() {
  return (
    <div className="flex h-screen w-full overflow-hidden bg-zinc-900">
      <div className="w-[65%] p-6">
        <ChangeGraph change={sampleChange} />
      </div>
      <div className="w-[35%] border-l border-zinc-700 bg-white">
        <TaskSidebar
          tasks={sampleChange.tasks}
          selectedTaskId={null}
        />
      </div>
    </div>
  )
}

export default App
