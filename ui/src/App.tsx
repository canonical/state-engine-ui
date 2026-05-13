import { useState } from "react";
import ChangeGraph from "./components/ChangeGraph";
import TaskSidebar from "./components/TaskSidebar";
import ChangeList from "./components/ChangeList";
import DebuggerControls from "./components/DebuggerControls";
import { useChange, useChangeList } from "./lib/useChange";
import { useStepper } from "./lib/useStepper";

function App() {
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);
  const [selectedChangeId, setSelectedChangeId] = useState<string | null>(null);
  const { changes, error: listError, loading: listLoading } = useChangeList();
  const { change, error: changeError, loading: changeLoading } = useChange(selectedChangeId);
  const stepper = useStepper(selectedChangeId, change?.ready ?? false);

  const selectedTask =
    change?.tasks.find((t) => t.id === selectedTaskId) ?? null;

  const showChangeList = !selectedChangeId;

  return (
    <div className="flex h-screen w-full overflow-hidden bg-zinc-50 dark:bg-zinc-900">
      {showChangeList ? (
        <div className="w-full h-full overflow-y-auto p-6 bg-white dark:bg-zinc-900">
          <h1 className="text-lg font-bold text-zinc-900 dark:text-zinc-100 mb-4">Changes</h1>
          <ChangeList
            changes={changes}
            selectedId={selectedChangeId}
            onSelect={setSelectedChangeId}
            error={listError}
            loading={listLoading}
          />
        </div>
      ) : (
        <>
          <div className="w-[65%] p-6">
            <div className="flex items-center gap-3 mb-4">
              <button
                type="button"
                onClick={() => { setSelectedChangeId(null); setSelectedTaskId(null) }}
                className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
              >
                &larr; Changes
              </button>
              <DebuggerControls
                steppingState={stepper.steppingState}
                stepping={stepper.stepping}
                noRunnableTasks={stepper.noRunnableTasks}
                error={stepper.error}
                changeReady={change?.ready ?? false}
                onStep={stepper.step}
                onContinue={stepper.continue}
                onPause={stepper.pause}
              />
              <h1 className="text-sm font-semibold text-gray-200">
                {change ? `${change.kind}: ${change.summary}` : `Change #${selectedChangeId}`}
              </h1>
              {changeLoading && <span className="text-xs text-gray-500">connecting...</span>}
            </div>
            {changeError && (
              <p className="text-red-400 text-sm mb-2">{changeError}</p>
            )}
            {change ? (
              <ChangeGraph
                change={change}
                selectedTaskId={selectedTaskId}
                steppingTaskId={stepper.lastSteppedTaskId}
                onSelectTask={setSelectedTaskId}
              />
            ) : !changeError ? (
              <p className="text-gray-400 text-sm">Loading...</p>
            ) : null}
          </div>
          <div className="w-[35%] border-l border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900">
            <TaskSidebar
              tasks={change?.tasks ?? []}
              selectedTaskId={selectedTaskId}
              selectedTask={selectedTask}
              onSelectTask={setSelectedTaskId}
              onClearSelection={() => setSelectedTaskId(null)}
            />
          </div>
        </>
      )}
    </div>
  );
}

export default App;
