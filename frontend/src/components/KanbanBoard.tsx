import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  DndContext,
  closestCorners,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { tasksApi } from '@/lib/api'
import type { Task, TaskStatus } from '@/types'
import KanbanCard from './KanbanCard'

const STATUS_COLS: TaskStatus[] = ['todo', 'in_progress', 'done']

interface KanbanBoardProps {
  tasks: Task[]
  projectId: string
  isOwner: boolean
  onEditTask: (task: Task) => void
}

export default function KanbanBoard({ tasks, projectId, isOwner, onEditTask }: KanbanBoardProps) {
  const qc = useQueryClient()
  const [activeId, setActiveId] = useState<string | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const updateTaskStatus = useMutation({
    mutationFn: ({ taskId, status }: { taskId: string; status: TaskStatus }) =>
      tasksApi.update(taskId, { status }),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ['project', projectId] })
    },
  })

  const deleteTask = useMutation({
    mutationFn: tasksApi.delete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['project', projectId] })
    },
  })

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)

    if (!over) return

    const activeTask = tasks.find((t) => t.id === active.id)
    const overStatus = over.id as TaskStatus

    if (activeTask && activeTask.status !== overStatus && STATUS_COLS.includes(overStatus)) {
      updateTaskStatus.mutate({ taskId: activeTask.id, status: overStatus })
    }
  }

  const columnsByStatus: Record<TaskStatus, Task[]> = {
    todo: tasks.filter((t) => t.status === 'todo'),
    in_progress: tasks.filter((t) => t.status === 'in_progress'),
    done: tasks.filter((t) => t.status === 'done'),
  }

  const columnLabels: Record<TaskStatus, string> = {
    todo: 'To Do',
    in_progress: 'In Progress',
    done: 'Done',
  }

  const columnColors: Record<TaskStatus, string> = {
    todo: 'bg-slate-50 dark:bg-slate-800/50',
    in_progress: 'bg-blue-50 dark:bg-blue-900/20',
    done: 'bg-green-50 dark:bg-green-900/20',
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragEnd={handleDragEnd}
      onDragStart={(event) => setActiveId(event.active.id as string)}
    >
      <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
        {STATUS_COLS.map((status) => (
          <div key={status} className="flex flex-col rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
            <div className={`${columnColors[status]} border-b border-slate-200 dark:border-slate-700 px-4 py-3`}>
              <div className="flex items-center justify-between">
                <h2 className="font-semibold text-slate-900 dark:text-white">{columnLabels[status]}</h2>
                <span className="inline-block rounded-full bg-slate-200 dark:bg-slate-600 px-2.5 py-0.5 text-xs font-medium text-slate-700 dark:text-slate-200">
                  {columnsByStatus[status].length}
                </span>
              </div>
            </div>
            <SortableContext
              items={columnsByStatus[status].map((t) => t.id)}
              strategy={verticalListSortingStrategy}
            >
              <div className="flex flex-1 flex-col gap-3 overflow-y-auto bg-white dark:bg-slate-900 p-4 min-h-96">
                {columnsByStatus[status].length === 0 ? (
                  <div className="flex items-center justify-center rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-600 py-12 text-center">
                    <p className="text-xs text-slate-400 dark:text-slate-500">No tasks</p>
                  </div>
                ) : (
                  columnsByStatus[status].map((task) => (
                    <KanbanCard
                      key={task.id}
                      task={task}
                      isActive={activeId === task.id}
                      isOwner={isOwner}
                      onEdit={() => onEditTask(task)}
                      onDelete={() => deleteTask.mutate(task.id)}
                    />
                  ))
                )}
              </div>
            </SortableContext>
          </div>
        ))}
      </div>
    </DndContext>
  )
}
