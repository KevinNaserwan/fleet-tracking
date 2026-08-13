"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ArrowClockwise,
  ArrowRight,
  Broadcast,
  Buildings,
  CheckCircle,
  ClockCountdown,
  Database,
  HardDrives,
  Key,
  List,
  MagnifyingGlass,
  MapTrifold,
  NavigationArrow,
  Package,
  ShieldCheck,
  SignOut,
  SlidersHorizontal,
  SquaresFour,
  Stack,
  Truck,
  UserCircle,
  UsersThree,
  WarningCircle,
  Wrench,
  X
} from "@phosphor-icons/react";

import { BrandMark } from "@/components/brand-mark";
import { DashboardSkeleton } from "@/components/dashboard-skeleton";
import { isCurrentUser, type CurrentUser } from "@/lib/contracts";

type SessionState =
  | { kind: "loading" }
  | { kind: "ready"; user: CurrentUser }
  | { kind: "expired"; requestId?: string }
  | { kind: "error"; message: string; requestId?: string };

interface LogoutPayload {
  data?: {
    loggedOut?: boolean;
    localSessionCleared?: boolean;
  };
  error?: {
    message?: string;
  };
}

type FleetStatus = "Bergerak" | "Berhenti" | "Siap" | "Perawatan" | "Offline";

interface FleetRecord {
  unit: string;
  plate: string;
  vehicle: string;
  driver: string;
  base: string;
  status: FleetStatus;
  activity: string;
}

const fleetRecords: FleetRecord[] = [
  { unit: "FLT-014", plate: "B 9342 KX", vehicle: "Hino Dutro 130 HD", driver: "Arif Nugroho", base: "Cakung", status: "Bergerak", activity: "07.42 WIB" },
  { unit: "FLT-021", plate: "B 8715 TXF", vehicle: "Isuzu Giga FVR", driver: "Dani Setiawan", base: "Tanjung Priok", status: "Berhenti", activity: "07.39 WIB" },
  { unit: "FLT-008", plate: "D 7813 AG", vehicle: "Fuso Fighter FM65", driver: "Rahmat Hidayat", base: "Bandung", status: "Perawatan", activity: "Jadwal bengkel" },
  { unit: "FLT-033", plate: "L 9421 UB", vehicle: "UD Trucks Quester", driver: "Bima Prakoso", base: "Surabaya", status: "Bergerak", activity: "07.35 WIB" },
  { unit: "FLT-019", plate: "B 9654 UIZ", vehicle: "Toyota Dyna 136 HT", driver: "Siti Maulida", base: "Bekasi", status: "Siap", activity: "Siap dispatch" },
  { unit: "FLT-027", plate: "H 8172 EQ", vehicle: "Hino Ranger FG", driver: "Fajar Maulana", base: "Semarang", status: "Offline", activity: "Belum tersambung" },
  { unit: "FLT-005", plate: "B 9128 FEV", vehicle: "Isuzu ELF NMR", driver: "Yoga Saputra", base: "Cibitung", status: "Berhenti", activity: "07.28 WIB" },
  { unit: "FLT-041", plate: "N 8732 KL", vehicle: "Fuso Canter FE74", driver: "Andi Kurniawan", base: "Malang", status: "Bergerak", activity: "07.24 WIB" }
];

const markers = [
  { unit: "014", left: "18%", top: "27%" },
  { unit: "021", left: "67%", top: "19%" },
  { unit: "033", left: "74%", top: "62%" },
  { unit: "041", left: "42%", top: "70%" }
];

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join("") || "OP";
}

function DashboardStateError({ state, retry }: { state: Extract<SessionState, { kind: "error" | "expired" }>; retry: () => void }) {
  const expired = state.kind === "expired";
  return (
    <main className="state-page">
      <div className="state-panel">
        <span className="state-icon" aria-hidden="true">
          {expired ? <Key size={26} weight="bold" /> : <WarningCircle size={26} weight="bold" />}
        </span>
        <p className="section-kicker">{expired ? "Sesi berakhir" : "Gangguan koneksi"}</p>
        <h1>{expired ? "Masuk kembali untuk melanjutkan." : "Ruang kontrol belum dapat dimuat."}</h1>
        <p>
          {expired
            ? "Token sesi tidak lagi valid dan telah dibersihkan dari browser."
            : state.message}
        </p>
        {state.requestId && <code className="request-id">Request ID: {state.requestId}</code>}
        {expired ? (
          <a className="primary-button" href="/login?next=%2Fdashboard">
            Masuk kembali <ArrowRight size={18} weight="bold" />
          </a>
        ) : (
          <button className="primary-button" type="button" onClick={retry}>
            <ArrowClockwise size={18} weight="bold" /> Coba lagi
          </button>
        )}
      </div>
    </main>
  );
}

export function DashboardClient() {
  const [session, setSession] = useState<SessionState>({ kind: "loading" });
  const [attempt, setAttempt] = useState(0);

  const reloadSession = useCallback(() => {
    setSession({ kind: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void (async () => {
      try {
        const response = await fetch("/api/session", {
          method: "GET",
          credentials: "same-origin",
          cache: "no-store",
          signal: controller.signal
        });
        const payload = (await response.json().catch(() => null)) as Record<string, unknown> | null;
        if (response.status === 401) {
          const error = payload?.error as { requestId?: string } | undefined;
          setSession({ kind: "expired", requestId: error?.requestId });
          return;
        }
        if (!response.ok) {
          const error = payload?.error as { message?: string; requestId?: string } | undefined;
          setSession({
            kind: "error",
            message: error?.message ?? "Layanan sesi mengembalikan respons yang tidak dapat digunakan.",
            requestId: error?.requestId
          });
          return;
        }
        const user = payload?.data;
        if (!isCurrentUser(user)) {
          setSession({ kind: "error", message: "Data profil tidak sesuai kontrak aplikasi." });
          return;
        }
        setSession({ kind: "ready", user });
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          return;
        }
        setSession({ kind: "error", message: "Browser tidak dapat menghubungi BFF sesi." });
      }
    })();
    return () => controller.abort();
  }, [attempt]);

  if (session.kind === "loading") {
    return <DashboardSkeleton />;
  }
  if (session.kind === "error" || session.kind === "expired") {
    return <DashboardStateError state={session} retry={reloadSession} />;
  }
  return <ControlRoom user={session.user} refreshIdentity={reloadSession} />;
}

function ControlRoom({ user, refreshIdentity }: { user: CurrentUser; refreshIdentity: () => void }) {
  const router = useRouter();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"Semua" | FleetStatus>("Semua");
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const mobileNavRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!mobileOpen) {
      return;
    }
    const previousOverflow = document.body.style.overflow;
    const previouslyFocused = document.activeElement instanceof HTMLElement ? document.activeElement : menuButtonRef.current;
    document.body.style.overflow = "hidden";
    mobileNavRef.current?.querySelector<HTMLElement>("a, button")?.focus();
    const handleDialogKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMobileOpen(false);
        return;
      }
      if (event.key !== "Tab" || !mobileNavRef.current) {
        return;
      }

      const focusable = Array.from(
        mobileNavRef.current.querySelectorAll<HTMLElement>(
          'a[href], button:not(:disabled), [tabindex]:not([tabindex="-1"])'
        )
      );
      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const activeElement = document.activeElement;
      if (event.shiftKey && (activeElement === first || !mobileNavRef.current.contains(activeElement))) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && (activeElement === last || !mobileNavRef.current.contains(activeElement))) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", handleDialogKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", handleDialogKeyDown);
      if (previouslyFocused?.isConnected) {
        previouslyFocused.focus();
      }
    };
  }, [mobileOpen]);

  const filteredFleet = useMemo(() => {
    const normalized = query.trim().toLocaleLowerCase("id-ID");
    return fleetRecords.filter((record) => {
      const matchesStatus = status === "Semua" || record.status === status;
      const searchable = `${record.unit} ${record.plate} ${record.vehicle} ${record.driver} ${record.base}`.toLocaleLowerCase("id-ID");
      return matchesStatus && (!normalized || searchable.includes(normalized));
    });
  }, [query, status]);

  const logout = useCallback(async () => {
    setLoggingOut(true);
    setLogoutError(null);
    try {
      const response = await fetch("/api/session/logout", {
        method: "POST",
        credentials: "same-origin",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: "{}"
      });
      const payload = (await response.json().catch(() => null)) as LogoutPayload | null;
      if (!response.ok || payload?.data?.loggedOut !== true || payload.data.localSessionCleared !== true) {
        setLogoutError(payload?.error?.message ?? "Sesi lokal belum dapat dihapus. Coba keluar kembali.");
        setMobileOpen(false);
        return;
      }
      router.push("/login");
      router.refresh();
    } catch {
      setLogoutError("Aplikasi tidak dapat menghubungi layanan sesi. Anda masih masuk.");
      setMobileOpen(false);
    } finally {
      setLoggingOut(false);
    }
  }, [router]);

  return (
    <div className="dashboard-layout">
      <DesktopSidebar user={user} loggingOut={loggingOut} logout={logout} />

      <header className="mobile-header" inert={mobileOpen ? true : undefined}>
        <BrandMark />
        <button
          ref={menuButtonRef}
          className="icon-button"
          type="button"
          onClick={() => setMobileOpen(true)}
          aria-label="Buka navigasi"
          aria-expanded={mobileOpen}
          aria-controls="mobile-navigation"
        >
          <List size={22} weight="bold" />
        </button>
      </header>

      {mobileOpen && (
        <>
          <button className="nav-scrim" type="button" tabIndex={-1} aria-hidden="true" onClick={() => setMobileOpen(false)} />
          <aside
            id="mobile-navigation"
            ref={mobileNavRef}
            className="mobile-drawer"
            role="dialog"
            aria-modal="true"
            aria-labelledby="mobile-navigation-title"
          >
            <div className="drawer-header">
              <BrandMark />
              <h2 className="sr-only" id="mobile-navigation-title">Navigasi utama</h2>
              <button className="icon-button" type="button" onClick={() => setMobileOpen(false)} aria-label="Tutup navigasi">
                <X size={21} weight="bold" />
              </button>
            </div>
            <NavItems close={() => setMobileOpen(false)} />
            <div className="drawer-user">
              <UserSummary user={user} />
              <button className="secondary-button full-width" type="button" onClick={logout} disabled={loggingOut}>
                <SignOut size={17} /> {loggingOut ? "Mengakhiri sesi..." : "Keluar"}
              </button>
            </div>
          </aside>
        </>
      )}

      <main className="dashboard-stage" id="main-content" inert={mobileOpen ? true : undefined}>
        {logoutError && (
          <div className="form-error logout-error" role="alert">
            <strong>Keluar gagal</strong>
            <span>{logoutError}</span>
          </div>
        )}
        <section className="operation-header" aria-labelledby="overview-title">
          <div>
            <p className="section-kicker">Milestone 1 / Identity control</p>
            <h1 id="overview-title">Ikhtisar operasi</h1>
            <p>Akses tenant aktif. Data armada dan peta masih berupa demonstrasi antarmuka.</p>
          </div>
          <button className="secondary-button" type="button" onClick={refreshIdentity}>
            <ArrowClockwise size={17} weight="bold" /> Muat ulang identitas
          </button>
        </section>

        <section className="readiness-ledger" aria-labelledby="readiness-title">
          <div className="ledger-heading">
            <h2 id="readiness-title">Kesiapan produk</h2>
            <span>Kontrak implementasi</span>
          </div>
          <div className="ledger-items">
            <div className="ledger-item is-current">
              <span>M1</span>
              <strong>Akses & identitas</strong>
              <small>Tersedia</small>
            </div>
            <div className="ledger-item">
              <span>M2</span>
              <strong>Data armada</strong>
              <small>Demo statis</small>
            </div>
            <div className="ledger-item">
              <span>M3</span>
              <strong>Pelacakan langsung</strong>
              <small>Belum tersedia</small>
            </div>
            <div className="ledger-item">
              <span>M4</span>
              <strong>Perawatan</strong>
              <small>Direncanakan</small>
            </div>
          </div>
        </section>

        <div className="primary-grid">
          <section className="map-panel" aria-labelledby="map-title">
            <div className="panel-heading">
              <div>
                <h2 id="map-title">Kanvas operasi</h2>
                <p>Representasi tata letak untuk kesiapan modul tracking.</p>
              </div>
              <span className="neutral-badge">Live tracking di M3</span>
            </div>
            <div
              className="ops-map"
              role="img"
              aria-label="Kanvas peta demonstrasi. Koordinat kendaraan langsung belum tersedia sampai Milestone 3."
            >
              <div className="map-grid-cells" aria-hidden="true">
                {Array.from({ length: 96 }, (_, index) => <span className="map-cell" key={index} />)}
              </div>
              <span className="contour contour-one" aria-hidden="true" />
              <span className="contour contour-two" aria-hidden="true" />
              <span className="road road-a" aria-hidden="true" />
              <span className="road road-b" aria-hidden="true" />
              <span className="road road-c" aria-hidden="true" />
              <span className="map-place place-a">CAKUNG</span>
              <span className="map-place place-b">PRIOK</span>
              <span className="map-place place-c">CIBITUNG</span>
              {markers.map((marker, index) => (
                <span
                  className={`map-vehicle${index === 0 ? " is-highlighted" : ""}`}
                  style={{ left: marker.left, top: marker.top }}
                  key={marker.unit}
                  aria-hidden="true"
                >
                  <Truck size={15} weight="fill" />
                  <small>{marker.unit}</small>
                </span>
              ))}
              <div className="map-notice">
                <MapTrifold size={21} aria-hidden="true" />
                <div>
                  <strong>Belum ada koordinat langsung</strong>
                  <span>Integrasi telematika dijadwalkan pada Milestone 3.</span>
                </div>
              </div>
            </div>
          </section>

          <aside className="health-rail" aria-labelledby="health-title">
            <div className="panel-heading rail-heading">
              <div>
                <h2 id="health-title">Kesehatan infrastruktur</h2>
                <p>Batas observabilitas antarmuka M1.</p>
              </div>
            </div>
            <div className="health-list">
              <HealthItem icon={<ShieldCheck size={19} />} label="Sesi BFF" detail="Terverifikasi lewat /me" state="Siap" active />
              <HealthItem icon={<Broadcast size={19} />} label="Identity API" detail="Merespons profil aktif" state="Terhubung" />
              <HealthItem icon={<Database size={19} />} label="PostgreSQL" detail="Tidak diekspos ke UI" state="Tidak dipantau" />
              <HealthItem icon={<HardDrives size={19} />} label="Redis" detail="Tidak diekspos ke UI" state="Tidak dipantau" />
              <HealthItem icon={<Stack size={19} />} label="RabbitMQ" detail="Tidak diekspos ke UI" state="Tidak dipantau" />
            </div>
            <p className="rail-disclosure">Rail ini tidak memanggil endpoint health dan tidak menyimpulkan status layanan langsung.</p>
          </aside>
        </div>

        <section className="fleet-section" id="fleet" aria-labelledby="fleet-title">
          <div className="fleet-heading">
            <div>
              <div className="heading-line">
                <h2 id="fleet-title">Armada prioritas</h2>
                <span className="demo-badge">Data demonstrasi</span>
              </div>
              <p>Delapan kendaraan statis untuk validasi produk. Endpoint armada dijadwalkan pada M2.</p>
            </div>
            <div className="fleet-controls">
              <label className="search-control">
                <span className="sr-only">Cari armada</span>
                <MagnifyingGlass size={17} aria-hidden="true" />
                <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Cari unit, pelat, pengemudi" />
              </label>
              <label className="select-control">
                <span className="sr-only">Filter status</span>
                <SlidersHorizontal size={17} aria-hidden="true" />
                <select value={status} onChange={(event) => setStatus(event.target.value as "Semua" | FleetStatus)}>
                  <option>Semua</option>
                  <option>Bergerak</option>
                  <option>Berhenti</option>
                  <option>Siap</option>
                  <option>Perawatan</option>
                  <option>Offline</option>
                </select>
              </label>
            </div>
          </div>

          {filteredFleet.length > 0 ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th scope="col">Unit</th>
                    <th scope="col">Kendaraan</th>
                    <th scope="col">Pengemudi</th>
                    <th scope="col">Basis</th>
                    <th scope="col">Status demo</th>
                    <th scope="col">Aktivitas demo</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredFleet.map((record) => (
                    <tr key={record.unit}>
                      <td data-label="Unit">
                        <strong className="unit-id">{record.unit}</strong>
                        <span className="plate">{record.plate}</span>
                      </td>
                      <td data-label="Kendaraan">{record.vehicle}</td>
                      <td data-label="Pengemudi">{record.driver}</td>
                      <td data-label="Basis">{record.base}</td>
                      <td data-label="Status demo"><span className={`fleet-status status-${record.status.toLowerCase()}`}>{record.status}</span></td>
                      <td data-label="Aktivitas demo"><span className="activity-cell"><ClockCountdown size={15} /> {record.activity}</span></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty-state">
              <MagnifyingGlass size={25} aria-hidden="true" />
              <strong>Tidak ada kendaraan demo yang cocok</strong>
              <span>Ubah kata pencarian atau kembalikan filter status.</span>
              <button className="secondary-button" type="button" onClick={() => { setQuery(""); setStatus("Semua"); }}>
                Atur ulang filter
              </button>
            </div>
          )}
        </section>

        <div className="secondary-grid">
          <section className="identity-panel" id="identity" aria-labelledby="identity-title">
            <div className="panel-heading">
              <div>
                <h2 id="identity-title">Profil & hak akses</h2>
                <p>Data langsung dari endpoint <code>/api/v1/me</code>.</p>
              </div>
              <span className="verified-label"><CheckCircle size={16} weight="fill" /> Terverifikasi</span>
            </div>
            <div className="profile-record">
              <span className="profile-avatar" aria-hidden="true">{initials(user.fullName)}</span>
              <div>
                <strong>{user.fullName}</strong>
                <span>{user.email}</span>
              </div>
              <dl>
                <div><dt>Tenant</dt><dd>{user.tenant.name}</dd></div>
                <div><dt>Zona waktu</dt><dd>{user.tenant.timezone}</dd></div>
                <div><dt>ID pengguna</dt><dd><code>{user.id}</code></dd></div>
              </dl>
            </div>
            <div className="access-grid">
              <div>
                <h3><UsersThree size={17} /> Peran</h3>
                {user.roles.length > 0 ? (
                  <div className="access-tags">{user.roles.map((role) => <code key={role}>{role}</code>)}</div>
                ) : (
                  <p className="inline-empty">Tidak ada peran efektif pada profil ini.</p>
                )}
              </div>
              <div>
                <h3><Key size={17} /> Izin efektif</h3>
                {user.permissions.length > 0 ? (
                  <div className="access-tags">{user.permissions.map((permission) => <code key={permission}>{permission}</code>)}</div>
                ) : (
                  <p className="inline-empty">Tidak ada izin efektif pada profil ini.</p>
                )}
              </div>
            </div>
          </section>

          <section className="activity-panel" aria-labelledby="activity-title">
            <div className="panel-heading">
              <div>
                <div className="heading-line">
                  <h2 id="activity-title">Alur aktivitas</h2>
                  <span className="demo-badge">Contoh statis</span>
                </div>
                <p>Format timeline untuk integrasi event berikutnya.</p>
              </div>
            </div>
            <ol className="activity-timeline">
              <TimelineItem icon={<ShieldCheck size={17} />} title="Sesi operator dibuat" detail="Contoh event autentikasi M1" />
              <TimelineItem icon={<NavigationArrow size={17} />} title="Unit FLT-014 memulai rute" detail="Contoh event tracking M3" />
              <TimelineItem icon={<Package size={17} />} title="Manifest FLT-033 diperbarui" detail="Contoh event operasi M2" />
              <TimelineItem icon={<Wrench size={17} />} title="FLT-008 masuk antrean bengkel" detail="Contoh event perawatan M4" />
            </ol>
          </section>
        </div>
      </main>
    </div>
  );
}

function DesktopSidebar({ user, loggingOut, logout }: { user: CurrentUser; loggingOut: boolean; logout: () => void }) {
  return (
    <aside className="desktop-sidebar">
      <div>
        <BrandMark />
        <div className="tenant-block">
          <Buildings size={17} aria-hidden="true" />
          <div><span>Tenant aktif</span><strong>{user.tenant.name}</strong></div>
        </div>
        <NavItems />
      </div>
      <div className="sidebar-user">
        <UserSummary user={user} />
        <button className="logout-button" type="button" onClick={logout} disabled={loggingOut} aria-label="Keluar dari sesi">
          <SignOut size={18} />
        </button>
      </div>
    </aside>
  );
}

function NavItems({ close }: { close?: () => void }) {
  return (
    <nav className="nav-items" aria-label="Navigasi utama">
      <p>Navigasi</p>
      <a className="nav-item active" href="#main-content" onClick={close}><SquaresFour size={18} weight="fill" /> Ringkasan</a>
      <a className="nav-item" href="#fleet" onClick={close}><Truck size={18} /> Armada <small>M2</small></a>
      <span className="nav-item disabled" aria-disabled="true"><MapTrifold size={18} /> Pelacakan <small>M3</small></span>
      <span className="nav-item disabled" aria-disabled="true"><Wrench size={18} /> Perawatan <small>M4</small></span>
      <p className="nav-section-label">Akun</p>
      <a className="nav-item" href="#identity" onClick={close}><UserCircle size={18} /> Profil & izin</a>
    </nav>
  );
}

function UserSummary({ user }: { user: CurrentUser }) {
  return (
    <div className="user-summary">
      <span className="user-avatar" aria-hidden="true">{initials(user.fullName)}</span>
      <div><strong>{user.fullName}</strong><span>{user.email}</span></div>
    </div>
  );
}

function HealthItem({ icon, label, detail, state, active = false }: { icon: React.ReactNode; label: string; detail: string; state: string; active?: boolean }) {
  return (
    <div className="health-item">
      <span className="health-icon" aria-hidden="true">{icon}</span>
      <div><strong>{label}</strong><span>{detail}</span></div>
      <small className={active ? "active-state" : ""}>{state}</small>
    </div>
  );
}

function TimelineItem({ icon, title, detail }: { icon: React.ReactNode; title: string; detail: string }) {
  return (
    <li>
      <span className="timeline-icon" aria-hidden="true">{icon}</span>
      <div><strong>{title}</strong><span>{detail}</span></div>
    </li>
  );
}
