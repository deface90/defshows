import { createTheme, type MantineColorsTuple } from '@mantine/core'

// ---------------------------------------------------------------------------
// Brand palette. Change these to re-skin the app. `orange` is the primary
// accent; `dark` drives the (near-black) dark scheme backgrounds.
// ---------------------------------------------------------------------------
export const brand = {
  accent: '#fd7e14', // primary orange
  black: '#0a0a0a',
}

const orange: MantineColorsTuple = [
  '#fff4e6',
  '#ffe8cc',
  '#ffd8a8',
  '#ffc078',
  '#ffa94d',
  '#ff922b',
  '#fd7e14', // 6 — light primary shade
  '#f76707', // 7 — dark primary shade
  '#e8590c',
  '#d9480f',
]

// Near-black dark scale (index 7 is the default body background in Mantine).
const dark: MantineColorsTuple = [
  '#c1c2c5',
  '#a6a7ab',
  '#909296',
  '#5c5f66',
  '#373a40',
  '#2c2e33',
  '#18191c',
  '#101113',
  '#0a0a0a',
  '#050505',
]

export const theme = createTheme({
  primaryColor: 'brand',
  primaryShade: { light: 6, dark: 7 },
  colors: {
    brand: orange,
    dark,
  },
  defaultRadius: 'md',
})
