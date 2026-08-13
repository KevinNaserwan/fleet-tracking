export default function LoginLoading() {
  return (
    <main className="login-loading" aria-busy="true" aria-label="Memuat halaman masuk">
      <section className="login-loading-context">
        <span className="skeleton-block skeleton-brand" />
        <span className="skeleton-block login-loading-title" />
        <span className="skeleton-block login-loading-copy" />
      </section>
      <section className="login-loading-form">
        <span className="skeleton-block skeleton-title" />
        <span className="skeleton-block skeleton-field" />
        <span className="skeleton-block skeleton-field" />
        <span className="skeleton-block skeleton-field" />
        <span className="skeleton-block skeleton-field" />
      </section>
    </main>
  );
}
