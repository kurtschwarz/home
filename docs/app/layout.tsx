import './styles.css'

import { Layout, Navbar } from 'nextra-theme-docs'
import { getPageMap } from 'nextra/page-map'

const navbar = (
  <Navbar
    logo={<b>Kurt's Homelab</b>}
    projectLink='https://github.com/kurtschwarz/home'
  />
)


export default async function RootLayout({ children }) {
  const pageMap = await getPageMap('/')

  return (
    <html
      lang="en"
      dir="ltr"
      suppressHydrationWarning
    >
      <head></head>
      <body>
        <Layout
          navbar={navbar}
          sidebar={{ defaultMenuCollapseLevel: 1, toggleButton: false }}
          pageMap={pageMap}
          darkMode
          nextThemes={{ forcedTheme: "dark", defaultTheme: "dark" }}
          feedback={{ content: null }}
          docsRepositoryBase='https://github.com/kurt/home/tree/main/docs'
        >
          {children}
        </Layout>
      </body>
    </html>
  )
}
