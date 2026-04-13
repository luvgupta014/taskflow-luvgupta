import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragOverEvent,
  DragStartEvent,
  DragOverlay,
  useDroppable,
} from '@dnd-kit/core'
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy, arrayMove } from '@dnd-kit/sortable'
import { tasksApi } from '@/lib/api'
import type { Task, TaskStatus } from '@/types'
import KanbanCard from './KanbanCard'

const STATUS_COLS: TaskStatus[] = ['todo', 'in_progress', 'done']

interface KanbanBoardProps {
  tasks: Task[]
  projectId: string
  isOwner: boolean
  memberMap: Record<string, string>
  onEditTask: (task: Task) => void
}

function DroppableColumn({ id, children, color, label, count }: {
  id: string
  children: React.ReactNode
  color: string
  label: string
  count: number
}) {
  const { setNodeRef, isOver } = useDroppable({ id })

  return (
    <div className="flex flex-col rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
      <div className={`${color} border-b border-slate-200 dark:border-slate-700 px-4 py-3`}>
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-slate-900 dark:text-white">{label}</h2>
          <span className="inline-block rounded-full bg-slate-200 dark:bg-slate-600 px-2.5 py-0.5 text-xs font-medium text-slate-700 dark:text-slate-200">
            {count}
          </span>
        </div>
      </div>
      <div
        ref={setNodeRef}
        className={`flex flex-1 flex-col gap-3 overflow-y-auto bg-white dark:bg-slate-900 p-4 min-h-[24rem] transition-colors ${
          isOver ? 'bg-brand-50 dark:bg-brand-900/10' : ''
        }`}
      >
        {children}
      </div>
    </div>
  )
}

export default function KanbanBoard({ tasks, projectId, isOwner, memberMap, onEditTask }: KanbanBoardProps) {
  const qc = useQueryClient()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [localTasks, setLocalTasks] = useState<Task[]>(tasks)

  // Sync with server data
  if (tasks !== localTasks && !activeId) {
    setLocalTasks(tasks)
  }

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const updateTask = useMutation({
    mutationFn: ({ taskId, status, order }: { taskId: string; status: TaskStatus; order?: number }) =>
      tasksApi.update(taskId, { status, ...(order !== undefined && { order }) }),
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

  const getColumn = (taskId: string): TaskStatus | null => {
    const task = localTasks.find((t) => t.id === taskId)
    return task ? task.status : null
  }

  const findColumnForDroppable = (id: string): TaskStatus | null => {
    if (STATUS_COLS.includes(id as TaskStatus)) return id as TaskStatus
    return getColumn(id)
  }

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }

  const handleDragOver = (event: DragOverEvent) => {
    const { active, over } = event
    if (!over) return

    const activeCol = findColumnForDroppable(active.id as string)
    const overCol = findColumnForDroppable(over.id as string)

    if (!activeCol || !overCol || activeCol === overCol) return

    // Move task to new column (optimistic)
    setLocalTasks((prev) =>
      prev.map((t) => t.id === active.id ? { ...t, status: overCol } : t)
    )
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)

    if (!over) return

    const activeTask = localTasks.find((t) => t.id === active.id)
    if (!activeTask) return

    const overCol = findColumnForDroppable(over.id as string)
    if (!overCol) return

    const columnTasks = localTasks.filter((t) => t.status === overCol)
    const overTask = localTasks.find((t) => t.id === over.id)

    if (overTask && activeTask.id !== overTask.id && activeTask.status === overTask.status) {
      // Reorder within column
      const oldIdx = columnTasks.findIndex((t) => t.id === active.id)
      const newIdx = columnTasks.findIndex((t) => t.id === over.id)
      if (oldIdx !== -1 && newIdx !== -1 && oldIdx !== newIdx) {
        const reordered = arrayMove(columnTasks, oldIdx, newIdx)
        setLocalTasks((prev) => {
          const rest = prev.filter((t) => t.status !== overCol)
          return [...rest, ...reordered]
        })
        updateTask.mutate({ taskId: activeTask.id, status: overCol, order: newIdx })
        return
      }
    }

    // Status change (cross-column or drop on column)
    const originalTask = tasks.find((t) => t.id === active.id)
    if (originalTask && originalTask.status !== overCol) {
      updateTask.mutate({ taskId: activeTask.id, status: overCol })
    }
  }

  const columnsByStatus: Record<TaskStatus, Task[]> = {
    todo: localTasks.filter((t) => t.status === 'todo'),
    in_progress: localTasks.filter((t) => t.status === 'in_progress'),
    done: localTasks.filter((t) => t.status === 'done'),
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

  const activeTask = activeId ? localTasks.find((t) => t.id === activeId) : null

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
        {STATUS_COLS.map((status) => (
          <DroppableColumn
            key={status}
            id={status}
            color={columnColors[status]}
            label={columnLabels[status]}
            count={columnsByStatus[status].length}
          >
            <SortableContext
              items={columnsByStatus[status].map((t) => t.id)}
              strategy={verticalListSortingStrategy}
            >
              {columnsByStatus[status].length === 0 ? (
                <div className="flex items-center justify-center rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-600 py-12 text-center">
                  <p className="text-xs text-slate-400 dark:text-slate-500">Drop tasks here</p>
                </div>
              ) : (
                columnsByStatus[status].map((task) => (
                  <KanbanCard
                    key={task.id}
                    task={task}
                    isActive={activeId === task.id}
                    isOwner={isOwner}
                    assigneeName={task.assignee_id ? memberMap[task.assignee_id] : undefined}
                    onEdit={() => onEditTask(task)}
                    onDelete={() => deleteTask.mutate(task.id)}
                  />
                ))
              )}
            </SortableContext>
          </DroppableColumn>
        ))}
      </div>

      <DragOverlay>
        {activeTask ? (
          <div className="rounded-lg border border-brand-300 bg-white p-4 shadow-xl opacity-90 dark:bg-slate-800 dark:border-brand-600 rotate-2">
            <p className="font-medium text-slate-900 dark:text-white text-sm">{activeTask.title}</p>
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  )
}
