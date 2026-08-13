"use client";

import { useState, type FormEvent } from "react";
import {
  Buildings,
  CircleNotch,
  EnvelopeSimple,
  Eye,
  EyeSlash,
  LockKey,
  SignIn
} from "@phosphor-icons/react";

interface LoginFormProps {
  redirectTo: string;
}

interface ApiError {
  error?: {
    message?: string;
    requestId?: string;
  };
}

export function LoginForm({ redirectTo }: LoginFormProps) {
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<{ message: string; requestId?: string } | null>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    const form = new FormData(event.currentTarget);
    try {
      const response = await fetch("/api/session/login", {
        method: "POST",
        credentials: "same-origin",
        cache: "no-store",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          tenant_slug: form.get("tenant_slug"),
          email: form.get("email"),
          password: form.get("password")
        })
      });
      const payload = (await response.json().catch(() => null)) as ApiError | null;
      if (!response.ok) {
        setError({
          message: payload?.error?.message ?? "Login gagal. Periksa koneksi dan coba lagi.",
          requestId: payload?.error?.requestId
        });
        return;
      }
      window.location.assign(redirectTo);
    } catch {
      setError({ message: "Aplikasi tidak dapat menghubungi layanan sesi." });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form className="login-form" onSubmit={submit} noValidate>
      {error && (
        <div className="form-error" role="alert">
          <strong>Akses ditolak</strong>
          <span>{error.message}</span>
          {error.requestId && <code>Request ID: {error.requestId}</code>}
        </div>
      )}

      <label className="field-group">
        <span>Tenant</span>
        <span className="field-control">
          <Buildings size={18} aria-hidden="true" />
          <input
            name="tenant_slug"
            type="text"
            autoComplete="organization"
            inputMode="text"
            placeholder="nusantara-logistics"
            pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
            maxLength={63}
            required
            disabled={submitting}
          />
        </span>
      </label>

      <label className="field-group">
        <span>Email kerja</span>
        <span className="field-control">
          <EnvelopeSimple size={18} aria-hidden="true" />
          <input
            name="email"
            type="email"
            autoComplete="username"
            placeholder="operator@perusahaan.co.id"
            maxLength={320}
            required
            disabled={submitting}
          />
        </span>
      </label>

      <label className="field-group">
        <span>Kata sandi</span>
        <span className="field-control">
          <LockKey size={18} aria-hidden="true" />
          <input
            name="password"
            type={showPassword ? "text" : "password"}
            autoComplete="current-password"
            placeholder="Minimal 8 karakter"
            minLength={8}
            maxLength={1024}
            required
            disabled={submitting}
          />
          <button
            className="password-toggle"
            type="button"
            onClick={() => setShowPassword((visible) => !visible)}
            aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
            disabled={submitting}
          >
            {showPassword ? <EyeSlash size={18} /> : <Eye size={18} />}
          </button>
        </span>
      </label>

      <button className="primary-button login-submit" type="submit" disabled={submitting}>
        {submitting ? <CircleNotch className="spinner" size={19} aria-hidden="true" /> : <SignIn size={19} weight="bold" />}
        {submitting ? "Memverifikasi..." : "Masuk ke kontrol"}
      </button>
    </form>
  );
}
