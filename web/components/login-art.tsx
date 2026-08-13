"use client";

import { Buildings, Truck } from "@phosphor-icons/react";

export function LoginArt() {
  return (
    <div className="login-art" aria-hidden="true">
      <div className="login-art-grid">
        {Array.from({ length: 48 }, (_, index) => <span key={index} />)}
      </div>
      <span className="login-contour contour-alpha" />
      <span className="login-contour contour-beta" />
      <span className="login-route route-alpha" />
      <span className="login-route route-beta" />
      <span className="login-depot depot-alpha"><Truck size={17} weight="fill" /></span>
      <span className="login-depot depot-beta"><Buildings size={17} weight="fill" /></span>
    </div>
  );
}
