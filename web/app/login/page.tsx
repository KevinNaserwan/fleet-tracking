import type { Metadata } from "next";

import { BrandMark } from "@/components/brand-mark";
import { LoginArt } from "@/components/login-art";
import { LoginForm } from "@/components/login-form";

export const metadata: Metadata = { title: "Masuk" };
export const dynamic = "force-dynamic";

const CONTROL_CHARACTER_PATTERN = /[\u0000-\u001f\u007f]/;

function safeRedirect(value: string | string[] | undefined): string {
  const candidate = Array.isArray(value) ? value[0] : value;
  if (!candidate) {
    return "/dashboard";
  }

  let decoded: string;
  try {
    decoded = decodeURIComponent(candidate);
  } catch {
    return "/dashboard";
  }

  const isUnsafe = (redirect: string) =>
    !redirect.startsWith("/") ||
    redirect.startsWith("//") ||
    redirect.includes("\\") ||
    CONTROL_CHARACTER_PATTERN.test(redirect);

  return isUnsafe(candidate) || isUnsafe(decoded) ? "/dashboard" : candidate;
}

export default async function LoginPage({ searchParams }: { searchParams: Promise<{ next?: string | string[] }> }) {
  const params = await searchParams;
  return (
    <main className="login-shell">
      <section className="login-context" aria-labelledby="login-context-title">
        <div className="login-context-top">
          <BrandMark />
          <span className="m1-label">M1 / Secure access</span>
        </div>
        <div className="login-context-copy">
          <p className="section-kicker">Pusat kendali armada</p>
          <h1 id="login-context-title">Satu akses untuk operasi yang terukur.</h1>
          <p>Masuk ke tenant perusahaan untuk melihat identitas, peran, dan kesiapan sistem Milestone 1.</p>
        </div>
        <div className="login-readiness" aria-label="Cakupan Milestone 1">
          <div><span>01</span><strong>Sesi BFF</strong><small>HttpOnly</small></div>
          <div><span>02</span><strong>Identitas tenant</strong><small>/api/v1/me</small></div>
          <div><span>03</span><strong>Hak akses</strong><small>Efektif</small></div>
        </div>
        <LoginArt />
      </section>

      <section className="login-entry" aria-labelledby="login-title">
        <div className="login-mobile-brand"><BrandMark /></div>
        <div className="login-entry-inner">
          <div className="login-heading">
            <span className="secure-mark" aria-hidden="true">ID</span>
            <div>
              <p className="section-kicker">Akses operator</p>
              <h2 id="login-title">Masuk ke ruang kontrol</h2>
            </div>
          </div>
          <p className="login-intro">Gunakan kredensial tenant yang diberikan administrator armada.</p>
          <LoginForm redirectTo={safeRedirect(params.next)} />
          <p className="session-note">Token disimpan dalam cookie HttpOnly dan tidak pernah diberikan ke JavaScript browser.</p>
        </div>
      </section>
    </main>
  );
}
