import { useEffect } from 'react'
import { useForm, Controller } from 'react-hook-form'
import { tasksApi } from '@/lib/api'
import type { Task, TaskStatus, TaskPriority } from '@/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

interface FormValues {
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  due_date: string
}

interface TaskModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
  task?: Task
  onSuccess: () => void
}

export function TaskModal({ open, onOpenChange, projectId, task, onSuccess }: TaskModalProps) {
  const isEdit = !!task

  const { register, handleSubmit, control, reset, formState: { errors, isSubmitting } } = useForm<FormValues>({
    defaultValues: {
      title: '',
      description: '',
      status: 'todo',
      priority: 'medium',
      due_date: '',
    },
  })

  useEffect(() => {
    if (task) {
      reset({
        title: task.title,
        description: task.description ?? '',
        status: task.status,
        priority: task.priority,
        due_date: task.due_date ?? '',
      })
    } else {
      reset({ title: '', description: '', status: 'todo', priority: 'medium', due_date: '' })
    }
  }, [task, reset, open])

  const onSubmit = async (data: FormValues) => {
    const payload = {
      ...data,
      description: data.description || undefined,
      due_date: data.due_date || undefined,
    }
    if (isEdit) {
      await tasksApi.update(task!.id, payload)
    } else {
      await tasksApi.create(projectId, payload)
    }
    onSuccess()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit task' : 'New task'}</DialogTitle>
          <DialogDescription>
            {isEdit ? 'Update task details below.' : 'Fill in the details for your new task.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" id="task-form">
          <div className="space-y-1.5">
            <Label htmlFor="task-title">Title</Label>
            <Input
              id="task-title"
              placeholder="What needs to be done?"
              error={errors.title?.message}
              {...register('title', { required: 'Title is required' })}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="task-desc">Description <span className="text-slate-400">(optional)</span></Label>
            <Input id="task-desc" placeholder="Add more context..." {...register('description')} />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Status</Label>
              <Controller
                name="status"
                control={control}
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger id="task-status"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="todo">To Do</SelectItem>
                      <SelectItem value="in_progress">In Progress</SelectItem>
                      <SelectItem value="done">Done</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </div>

            <div className="space-y-1.5">
              <Label>Priority</Label>
              <Controller
                name="priority"
                control={control}
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger id="task-priority"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="low">Low</SelectItem>
                      <SelectItem value="medium">Medium</SelectItem>
                      <SelectItem value="high">High</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="task-due">Due date <span className="text-slate-400">(optional)</span></Label>
            <Input id="task-due" type="date" {...register('due_date')} />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" loading={isSubmitting} id="task-submit">
              {isEdit ? 'Save changes' : 'Create task'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
