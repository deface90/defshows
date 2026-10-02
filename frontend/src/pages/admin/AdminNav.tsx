import { Group } from '@mantine/core'
import { NavLink } from 'react-router-dom'

const links = [
  { to: '/admin', label: 'Студии озвучки', end: true },
  { to: '/admin/reports', label: 'Жалобы' },
]

/** AdminNav is the sub-navigation shared across admin pages. */
export function AdminNav() {
  return (
    <Group gap="lg">
      {links.map((l) => (
        <NavLink
          key={l.to}
          to={l.to}
          end={l.end}
          className={({ isActive }) => `app-navlink${isActive ? ' active' : ''}`}
        >
          {l.label}
        </NavLink>
      ))}
    </Group>
  )
}
