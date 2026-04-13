import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Edit2, Trash2 } from 'lucide-react'
import { Badge, priorityLabel } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { Task } from '@/types'
import { formatDate, isOverdue, cn } from '@/lib/utils'

interface KanbanCardProps {
  task: Task
  isActive: boolean
  isOwner: boolean
  onEdit: () => void
  onDelete: () => void
}

export default function KanbanCard({ task, isActive, isOwner, onEdit, onDelete }: KanbanCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: task.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition-all dark:border-slate-700 dark:bg-slate-800',
        isDragging && 'shadow-lg ring-2 ring-brand-500',
        isActive && 'ring-2 ring-brand-500'
      )}
    >
      <div className="flex items-start gap-2">
        <button
          {...attributes}
          {...listeners}
          className="mt-1 cursor-grab active:cursor-grabbing text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 flex-shrink-0"
          title="Drag to reorder"
        >
          <GripVertical className="h-4 w-4" />
        </button>
        <div className="flex-1 min-w-0">
          <p className={cn(
            'font-medium text-slate-900 dark:text-white break-words',
            task.status === 'done' && 'line-through text-slate-400 dark:text-slate-500'
          )}>
            {task.title}
          </p>
          {task.description && (
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400 line-clamp-2">{task.description}</p>
          )}
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <Badge variant={task.priority} className="text-xs">{priorityLabel[task.priority]}</Badge>
            {task.due_date && (
              <span className={cn(
                'text-xs font-medium',
                isOverdue(task.due_date) && task.status !== 'done'
                  ? 'text-red-600 dark:text-red-400'
                  : 'text-slate-500 dark:text-slate-400'
              )}>
                {formatDate(task.due_date)}
              </span>
            )}
          </div>
        </div>
      </div>
      {isOwner && (
        <div className="mt-3 flex gap-2 justify-end">
          <Button
            size="sm"
            variant="ghost"
            onClick={onEdit}
            className="h-6 w-6 p-0"
          >
            <Edit2 className="h-3 w-3" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={onDelete}
            className="h-6 w-6 p-0 text-red-600 hover:text-red-700 dark:text-red-400"
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
      )}
    </div>
  )
}
