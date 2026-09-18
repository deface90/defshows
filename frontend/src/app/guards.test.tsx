import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { useAuthStore } from '@/shared/auth/authStore'
import { RequireAuth, RequireRole } from './guards'

function setup() {
  return render(
    <MemoryRouter initialEntries={['/secret']}>
      <Routes>
        <Route element={<RequireAuth />}>
          <Route path="/secret" element={<div>secret content</div>} />
        </Route>
        <Route path="/login" element={<div>login page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('RequireAuth', () => {
  afterEach(() => useAuthStore.getState().clear())

  it('redirects guests to /login', () => {
    setup()
    expect(screen.getByText('login page')).toBeInTheDocument()
  })

  it('renders protected content for authenticated users', () => {
    useAuthStore.getState().setSession('acc', 'ref', {
      id: 1,
      display_name: 'A',
      role: 'user',
      timezone: 'UTC',
    })
    setup()
    expect(screen.getByText('secret content')).toBeInTheDocument()
  })
})

function setupRole() {
  return render(
    <MemoryRouter initialEntries={['/admin']}>
      <Routes>
        <Route element={<RequireRole role="admin" />}>
          <Route path="/admin" element={<div>admin content</div>} />
        </Route>
        <Route path="/" element={<div>home page</div>} />
        <Route path="/login" element={<div>login page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('RequireRole', () => {
  afterEach(() => useAuthStore.getState().clear())

  it('redirects guests to /login', () => {
    setupRole()
    expect(screen.getByText('login page')).toBeInTheDocument()
  })

  it('redirects non-admin users home', () => {
    useAuthStore.getState().setSession('acc', 'ref', {
      id: 1,
      display_name: 'A',
      role: 'user',
      timezone: 'UTC',
    })
    setupRole()
    expect(screen.getByText('home page')).toBeInTheDocument()
  })

  it('renders content for admins', () => {
    useAuthStore.getState().setSession('acc', 'ref', {
      id: 2,
      display_name: 'Admin',
      role: 'admin',
      timezone: 'UTC',
    })
    setupRole()
    expect(screen.getByText('admin content')).toBeInTheDocument()
  })
})
