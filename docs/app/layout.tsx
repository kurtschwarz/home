import './styles.css'

import { Layout, Navbar } from 'nextra-theme-docs'
import { getPageMap } from 'nextra/page-map'

const Logo = () => {
  return (
    <div className="x:flex x:items-center x:text-md">
      <svg viewBox="0 0 24 24" stroke="currentColor" fill="none" strokeWidth="2" height="18" className="x:shrink-0 x:rounded-sm x:p-0.5 x:hover:bg-gray-800/5 x:dark:hover:bg-gray-100/5 x:motion-reduce:transition-none x:origin-center x:transition-all x:rtl:-rotate-180">
        <path d="M9 5l7 7-7 7" strokeLinecap="round" strokeLinejoin="round"></path>
      </svg> docs@kurtainerd.io:~$ <div className='cursor' />
    </div>
  )
}

const navbar = (
  <Navbar
    logo={<Logo />}
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
