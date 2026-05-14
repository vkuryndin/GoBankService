import { Outlet } from 'react-router-dom'

export function PublicLayout() {
  return (
    <main className="publicPage">
      <section className="publicShell">
        <div className="publicBrand">
          <div className="brandMark">GB</div>
          <div>
            <strong>Go Bank</strong>
            <span>REST API frontend</span>
          </div>
        </div>

        <Outlet />
      </section>
    </main>
  )
}
