import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { isAuthenticated } from '@/lib/auth'

export function ProtectedRoute() {
  const location = useLocation()

  if (!isAuthenticated()) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  return <Outlet />
}

export function PublicOnlyRoute() {
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}
