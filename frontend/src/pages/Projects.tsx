import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, FolderOpen, Loader2 } from 'lucide-react'
import { projectsApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { formatDate } from '@/lib/utils'
import { Navbar } from '@/components/Navbar'

export default function Projects() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [nameError, setNameError] = useState('')

  const { data: projects = [], isLoading, isError } = useQuery({
    queryKey: ['projects'],
    queryFn: projectsApi.list,
  })

  const create = useMutation({
    mutationFn: projectsApi.create,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['projects'] })
      setOpen(false)
      setName('')
      setDescription('')
    },
  })

  const handleCreate = () => {
    if (!name.trim()) { setNameError('Name is required'); return }
    setNameError('')
    create.mutate({ name: name.trim(), description: description.trim() || undefined })
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <Navbar />

      <main className="mx-auto max-w-6xl px-4 py-8">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Projects</h1>
            <p className="mt-0.5 text-sm text-slate-500 dark:text-slate-400">
              {projects.length} project{projects.length !== 1 ? 's' : ''}
            </p>
          </div>

          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button id="new-project-btn">
                <Plus className="h-4 w-4" />
                New project
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create project</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="proj-name">Name</Label>
                  <Input
                    id="proj-name"
                    placeholder="Website Redesign"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    error={nameError}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="proj-desc">Description <span className="text-slate-400">(optional)</span></Label>
                  <Input
                    id="proj-desc"
                    placeholder="What is this project about?"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </div>
                <div className="flex justify-end gap-2 pt-2">
                  <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
                  <Button onClick={handleCreate} loading={create.isPending} id="create-project-submit">
                    Create
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>

        {isLoading && (
          <div className="flex items-center justify-center py-20">
            <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
          </div>
        )}

        {isError && (
          <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-center text-sm text-red-600 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
            Failed to load projects. Please refresh.
          </div>
        )}

        {!isLoading && !isError && projects.length === 0 && (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white py-20 dark:border-slate-700 dark:bg-slate-900">
            <FolderOpen className="mb-3 h-10 w-10 text-slate-300 dark:text-slate-600" />
            <p className="font-medium text-slate-500 dark:text-slate-400">No projects yet</p>
            <p className="mt-1 text-sm text-slate-400 dark:text-slate-500">Create your first project to get started</p>
          </div>
        )}

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((p) => (
            <Link
              key={p.id}
              to={`/projects/${p.id}`}
              id={`project-${p.id}`}
              className="group flex flex-col rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition-all hover:border-brand-300 hover:shadow-md dark:border-slate-800 dark:bg-slate-900 dark:hover:border-brand-700"
            >
              <div className="mb-3 flex items-start justify-between gap-2">
                <h2 className="font-semibold text-slate-900 group-hover:text-brand-600 dark:text-white dark:group-hover:text-brand-400 line-clamp-2">
                  {p.name}
                </h2>
              </div>
              {p.description && (
                <p className="mb-3 text-sm text-slate-500 dark:text-slate-400 line-clamp-2 flex-1">
                  {p.description}
                </p>
              )}
              {!p.description && <div className="flex-1" />}
              <p className="text-xs text-slate-400 dark:text-slate-500">
                Created {formatDate(p.created_at)}
              </p>
            </Link>
          ))}
        </div>
      </main>
    </div>
  )
}
