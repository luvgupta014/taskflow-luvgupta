import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Loader2 } from 'lucide-react'
import { projectsApi } from '@/lib/api'
import { Navbar } from '@/components/Navbar'
import { Badge, statusLabel } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

export default function ProjectStats() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: project } = useQuery({
    queryKey: ['project', id],
    queryFn: () => projectsApi.get(id!),
    enabled: !!id,
  })

  const { data: stats, isLoading, isError } = useQuery({
    queryKey: ['project-stats', id],
    queryFn: () => projectsApi.stats(id!),
    enabled: !!id,
  })

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <Navbar />
      <main className="mx-auto max-w-3xl px-4 py-8">
        <button
          onClick={() => navigate(`/projects/${id}`)}
          className="mb-6 flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 transition-colors"
        >
          <ArrowLeft className="h-4 w-4" /> {project?.name ?? 'Project'}
        </button>

        <h1 className="mb-6 text-2xl font-bold text-slate-900 dark:text-white">Stats</h1>

        {isLoading && (
          <div className="flex justify-center py-20">
            <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
          </div>
        )}

        {isError && (
          <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center text-sm text-red-600 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
            Failed to load stats.
          </div>
        )}

        {stats && (
          <div className="space-y-6">
            <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
              <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                By Status
              </h2>
              <div className="grid grid-cols-3 gap-4">
                {(['todo', 'in_progress', 'done'] as const).map((s) => (
                  <div key={s} className="flex flex-col items-center gap-2 rounded-xl border border-slate-100 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-800/50">
                    <span className="text-3xl font-bold text-slate-900 dark:text-white">
                      {stats.by_status[s] ?? 0}
                    </span>
                    <Badge variant={s}>{statusLabel[s]}</Badge>
                  </div>
                ))}
              </div>
            </div>

            {Object.keys(stats.by_assignee).length > 0 && (
              <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900">
                <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                  By Assignee
                </h2>
                <div className="space-y-3">
                  {Object.entries(stats.by_assignee).map(([uid, stat]) => (
                    <div key={uid} className="flex items-center justify-between rounded-lg bg-slate-50 px-4 py-3 dark:bg-slate-800/50">
                      <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{stat.name}</span>
                      <span className="text-sm font-bold text-brand-600">{stat.count} task{stat.count !== 1 ? 's' : ''}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {Object.keys(stats.by_assignee).length === 0 && (
              <p className="text-center text-sm text-slate-400 dark:text-slate-500">No assignees yet.</p>
            )}

            <div className="flex justify-center">
              <Button variant="outline" onClick={() => navigate(`/projects/${id}`)}>
                Back to project
              </Button>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
