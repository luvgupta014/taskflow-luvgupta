import axios from 'axios'
import type { AuthResponse, Project, Task, ProjectStats } from '@/types'

const API_BASE = import.meta.env.VITE_API_URL ?? '/api'

export const http = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('tf_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('tf_token')
      localStorage.removeItem('tf_user')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export const authApi = {
  register: (data: { name: string; email: string; password: string }) =>
    http.post<AuthResponse>('/auth/register', data).then((r) => r.data),

  login: (data: { email: string; password: string }) =>
    http.post<AuthResponse>('/auth/login', data).then((r) => r.data),
}

export const projectsApi = {
  list: () =>
    http.get<{ projects: Project[] }>('/projects').then((r) => r.data.projects),

  get: (id: string) =>
    http.get<Project>(`/projects/${id}`).then((r) => r.data),

  create: (data: { name: string; description?: string }) =>
    http.post<Project>('/projects', data).then((r) => r.data),

  update: (id: string, data: { name?: string; description?: string }) =>
    http.patch<Project>(`/projects/${id}`, data).then((r) => r.data),

  delete: (id: string) =>
    http.delete(`/projects/${id}`),

  stats: (id: string) =>
    http.get<ProjectStats>(`/projects/${id}/stats`).then((r) => r.data),
}

export const tasksApi = {
  list: (projectId: string, params?: { status?: string; assignee?: string }) =>
    http
      .get<{ tasks: Task[] }>(`/projects/${projectId}/tasks`, { params })
      .then((r) => r.data.tasks),

  create: (projectId: string, data: Partial<Task>) =>
    http.post<Task>(`/projects/${projectId}/tasks`, data).then((r) => r.data),

  update: (id: string, data: Partial<Task>) =>
    http.patch<Task>(`/tasks/${id}`, data).then((r) => r.data),

  delete: (id: string) =>
    http.delete(`/tasks/${id}`),
}
