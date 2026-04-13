import { Link, useNavigate } from 'react-router-dom'
import { LogOut, Moon, Sun, LayoutDashboard } from 'lucide-react'
import { useAuthStore } from '@/store/auth'
import { Button } from '@/components/ui/button'
import { useDarkMode } from '@/hooks/useDarkMode'

export function Navbar() {
  const { user, logout } = useAuthStore()
  const navigate = useNavigate()
  const { dark, toggle } = useDarkMode()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/80 backdrop-blur dark:border-slate-800 dark:bg-slate-950/80">
      <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
        <Link
          to="/projects"
          className="flex items-center gap-2 font-semibold text-slate-900 dark:text-white hover:opacity-80 transition-opacity"
        >
          <LayoutDashboard className="h-5 w-5 text-brand-600" />
          <span>TaskFlow</span>
        </Link>

        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon" onClick={toggle} aria-label="Toggle theme">
            {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </Button>

          {user && (
            <>
              <span className="hidden text-sm text-slate-600 dark:text-slate-400 sm:block">
                {user.name}
              </span>
              <Button variant="ghost" size="icon" onClick={handleLogout} aria-label="Logout">
                <LogOut className="h-4 w-4" />
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  )
}
