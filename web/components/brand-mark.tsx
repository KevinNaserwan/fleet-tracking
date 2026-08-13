"use client";

import { Truck } from "@phosphor-icons/react";

interface BrandMarkProps {
  compact?: boolean;
}

export function BrandMark({ compact = false }: BrandMarkProps) {
  return (
    <div className="brand-mark" aria-label="Fleet Tracking">
      <span className="brand-symbol" aria-hidden="true">
        <Truck size={20} weight="fill" />
      </span>
      {!compact && (
        <span className="brand-copy">
          <strong>FLEET</strong>
          <span>TRACKING</span>
        </span>
      )}
    </div>
  );
}
