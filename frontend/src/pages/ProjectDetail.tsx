import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Loader2, Trash2, ArrowLeft, BarChart2 } from 'lucide-react'
import { projectsApi, tasksApi } from '@/lib/api'
import { useAuthStore } from '@/store/auth'
import type { Task, TaskStatus } from '@/types'
import { Button } from '@/components/ui/button'
import { Badge, statusLabel, priorityLabel } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Navbar } from '@/components/Navbar'
import { TaskModal } from '@/components/TaskModal'
import KanbanBoard from '@/components/KanbanBoard'
import { formatDate, isOverdue, cn } from '@/lib/utils'

export default function ProjectDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { user } = useAuthStore()

  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [taskModalOpen, setTaskModalOpen] = useState(false)
  const [editingTask, setEditingTask] = useState<Task | undefined>()
  const [viewMode, setViewMode] = useState<'kanban' | 'list'>('kanban')

  const { data: project, isLoading, isError } = useQuery({
    queryKey: ['project', id],
    queryFn: () => projectsApi.get(id!),
    enabled: !!id,
  })

  const deleteProject = useMutation({
    mutationFn: () => projectsApi.delete(id!),
    onSuccess: () => navigate('/projects'),
  })

  const updateTaskStatus = useMutation({
    mutationFn: ({ taskId, status }: { taskId: string; status: TaskStatus }) =>
      tasksApi.update(taskId, { status }),
    onMutate: async ({ taskId, status }) => {
      await qc.cancelQueries({ queryKey: ['project', id] })
      const prev = qc.getQueryData(['project', id])
      qc.setQueryData(['project', id], (old: typeof project) => {
        if (!old) return old
        return { ...old, tasks: old.tasks?.map((t) => t.id === taskId ? { ...t, status } : t) }
      })
      return { prev }
    },
    onError: (_err, _vars, ctx) => {
      qc.setQueryData(['project', id], ctx?.prev)
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ['project', id] })
    },
  })

  const deleteTask = useMutation({
    mutationFn: tasksApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['project', id] }),
  })

  const filtered = (project?.tasks ?? []).filter(
    (t) => statusFilter === 'all' || t.status === statusFilter
  )

  const isOwner = project?.owner_id === user?.id

  if (isLoading) return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <Navbar />
      <div className="flex items-center justify-center py-32">
        <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
      </div>
    </div>
  )

  if (isError || !project) return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <Navbar />
      <div className="mx-auto max-w-6xl px-4 py-16 text-center">
        <p className="text-slate-500">Project not found or you don&apos;t have access.</p>
        <Button variant="link" onClick={() => navigate('/projects')} className="mt-2">← Back to projects</Button>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <Navbar />

      <main className="mx-auto max-w-6xl px-4 py-8">
        <button
          onClick={() => navigate('/projects')}
          className="mb-6 flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 transition-colors"
        >
          <ArrowLeft className="h-4 w-4" /> Projects
        </button>

        <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{project.name}</h1>
            {project.description && (
              <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{project.description}</p>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {viewMode === 'list' && (
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-36" id="status-filter">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="todo">To Do</SelectItem>
                  <SelectItem value="in_progress">In Progress</SelectItem>
                  <SelectItem value="done">Done</SelectItem>
                </SelectContent>
              </Select>
            )}

            <div className="flex items-center gap-1 border border-slate-200 dark:border-slate-700 rounded-lg p-1">
              <Button
                variant={viewMode === 'kanban' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('kanban')}
                className="h-7 px-3 text-xs"
              >
                Kanban
              </Button>
              <Button
                variant={viewMode === 'list' ? 'default' : 'ghost'}
                size="sm"
                onClick={() => setViewMode('list')}
                className="h-7 px-3 text-xs"
              >
                List
              </Button>
            </div>

            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate(`/projects/${id}/stats`)}
              id="stats-btn"
            >
              <BarChart2 className="h-4 w-4" /> Stats
            </Button>

            {isOwner && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => { if (confirm('Delete this project and all its tasks?')) deleteProject.mutate() }}
                loading={deleteProject.isPending}
                id="delete-project-btn"
              >
                <Trash2 className="h-4 w-4" /> Delete
              </Button>
            )}

            <Button
              onClick={() => { setEditingTask(undefined); setTaskModalOpen(true) }}
              id="add-task-btn"
            >
              <Plus className="h-4 w-4" /> Add task
            </Button>
          </div>
        </div>

        {viewMode === 'kanban' && (project?.tasks ?? []).length > 0 ? (
          <KanbanBoard
            tasks={project.tasks || []}
            projectId={id!}
            isOwner={isOwner}
            onEditTask={(task) => { setEditingTask(task); setTaskModalOpen(true) }}
          />
        ) : viewMode === 'kanban' ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white py-20 dark:border-slate-700 dark:bg-slate-900">
            <p className="font-medium text-slate-500 dark:text-slate-400">No tasks yet</p>
            <p className="mt-1 text-sm text-slate-400 dark:text-slate-500">Add your first task to get started</p>
          </div>
        ) : null}

        {viewMode === 'list' && (
          <>
            {filtered.length === 0 && (
              <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white py-20 dark:border-slate-700 dark:bg-slate-900">
                <p className="font-medium text-slate-500 dark:text-slate-400">No tasks here</p>
                <p className="mt-1 text-sm text-slate-400 dark:text-slate-500">
                  {statusFilter === 'all' ? 'Add your first task to get started' : 'Try a different filter'}
                </p>
              </div>
            )}

            <div className="space-y-2">
              {filtered.map((task) => (
                <div
                  key={task.id}
                  id={`task-${task.id}`}
                  className="flex flex-wrap items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 shadow-sm transition-shadow hover:shadow-md dark:border-slate-800 dark:bg-slate-900"
                >
                  <Select
                    value={task.status}
                    onValueChange={(v) => updateTaskStatus.mutate({ taskId: task.id, status: v as TaskStatus })}
                  >
                    <SelectTrigger className="h-7 w-32 text-xs" id={`status-${task.id}`}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {['todo', 'in_progress', 'done'].map((s) => (
                        <SelectItem key={s} value={s}>{statusLabel[s as TaskStatus]}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  <div className="flex-1 min-w-0">
                    <p className={cn(
                      'font-medium text-slate-900 dark:text-white truncate',
                      task.status === 'done' && 'line-through text-slate-400 dark:text-slate-500'
                    )}>
                      {task.title}
                    </p>
                    {task.description && (
                      <p className="mt-0.5 text-xs text-slate-400 dark:text-slate-500 truncate">{task.description}</p>
                    )}
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant={task.priority}>{priorityLabel[task.priority]}</Badge>
                    {task.due_date && (
                      <span className={cn(
                        'text-xs',
                        isOverdue(task.due_date) && task.status !== 'done'
                          ? 'text-red-500 font-medium'
                          : 'text-slate-400 dark:text-slate-500'
                      )}>
                        {formatDate(task.due_date)}
                      </span>
                    )}
                  </div>

                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => { setEditingTask(task); setTaskModalOpen(true) }}
                      id={`edit-task-${task.id}`}
                    >
                      Edit
                    </Button>
                    {isOwner && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => deleteTask.mutate(task.id)}
                        className="text-red-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                        id={`delete-task-${task.id}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

      </main>

      <TaskModal
        open={taskModalOpen}
        onOpenChange={setTaskModalOpen}
        projectId={id!}
        task={editingTask}
        onSuccess={() => qc.invalidateQueries({ queryKey: ['project', id] })}
      />
    </div>
  )
}
