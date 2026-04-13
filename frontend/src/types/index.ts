export interface User {
  id: string
  name: string
  email: string
}

export interface Project {
  id: string
  name: string
  description?: string
  owner_id: string
  created_at: string
  tasks?: Task[]
}

export type TaskStatus = 'todo' | 'in_progress' | 'done'
export type TaskPriority = 'low' | 'medium' | 'high'

export interface Task {
  id: string
  title: string
  description?: string
  status: TaskStatus
  priority: TaskPriority
  project_id: string
  assignee_id?: string | null
  due_date?: string
  order?: number
  created_by?: string
  created_at: string
  updated_at: string
}

export interface Member {
  id: string
  name: string
  email: string
}

export interface ProjectStats {
  by_status: Record<string, number>
  by_assignee: Record<string, { name: string; count: number }>
}

export interface AuthResponse {
  token: string
  user: User
}

export interface ApiError {
  error: string
  fields?: Record<string, string>
}
