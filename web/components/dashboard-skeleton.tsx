export function DashboardSkeleton() {
  return (
    <main className="dashboard-skeleton" aria-busy="true" aria-label="Memuat ruang kontrol">
      <aside className="skeleton-sidebar">
        <span className="skeleton-block skeleton-brand" />
        <span className="skeleton-block skeleton-nav" />
        <span className="skeleton-block skeleton-nav" />
        <span className="skeleton-block skeleton-nav" />
      </aside>
      <section className="skeleton-stage">
        <div className="skeleton-heading">
          <span className="skeleton-block skeleton-title" />
          <span className="skeleton-block skeleton-copy" />
        </div>
        <span className="skeleton-block skeleton-strip" />
        <div className="skeleton-grid">
          <span className="skeleton-block skeleton-map" />
          <span className="skeleton-block skeleton-rail" />
        </div>
        <span className="skeleton-block skeleton-table" />
      </section>
    </main>
  );
}
