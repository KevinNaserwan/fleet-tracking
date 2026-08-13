"use client";

import { ArrowClockwise, WarningCircle } from "@phosphor-icons/react";

export default function DashboardError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <main className="state-page">
      <div className="state-panel">
        <span className="state-icon" aria-hidden="true"><WarningCircle size={26} weight="bold" /></span>
        <p className="section-kicker">Kesalahan antarmuka</p>
        <h1>Ruang kontrol tidak dapat dirender.</h1>
        <p>Data sensitif tidak ditampilkan. Coba muat ulang batas halaman ini.</p>
        <button className="primary-button" type="button" onClick={reset}>
          <ArrowClockwise size={18} weight="bold" /> Muat ulang
        </button>
      </div>
    </main>
  );
}
