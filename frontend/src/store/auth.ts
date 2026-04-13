import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User } from '@/types'

interface AuthState {
  user: User | null
  token: string | null
  setAuth: (user: User, token: string) => void
  logout: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,

      setAuth: (user, token) => {
        localStorage.setItem('tf_token', token)
        localStorage.setItem('tf_user', JSON.stringify(user))
        set({ user, token })
      },

      logout: () => {
        localStorage.removeItem('tf_token')
        localStorage.removeItem('tf_user')
        set({ user: null, token: null })
      },

      isAuthenticated: () => !!get().token,
    }),
    {
      name: 'taskflow-auth',
      partialize: (s) => ({ user: s.user, token: s.token }),
    }
  )
)
